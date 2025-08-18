package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/odnielgonzalez/bedrock-manager-controller/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

var (
	version = "dev"
	// Custom scheme with our types registered
	customScheme = runtime.NewScheme()
)

func init() {
	// Add the default Kubernetes types to our scheme
	_ = scheme.AddToScheme(customScheme)

	// Register our custom types for both versioned and internal versions
	gv := schema.GroupVersion{Group: "bedrock.aws.example.com", Version: "v1"}
	customScheme.AddKnownTypes(gv,
		&controller.BedrockResource{},
		&controller.BedrockResourceList{},
	)

	// Register for internal version
	gvInternal := schema.GroupVersion{Group: "bedrock.aws.example.com", Version: runtime.APIVersionInternal}
	customScheme.AddKnownTypes(gvInternal,
		&controller.BedrockResource{},
		&controller.BedrockResourceList{},
	)

	metav1.AddToGroupVersion(customScheme, gv)
}

// ControllerConfig holds the configuration for the controller
type ControllerConfig struct {
	KubeConfig     string
	MetricsAddr    string
	LeaderElection bool
	Region         string
	Workers        int
}

func main() {
	klog.InitFlags(nil)

	config := &ControllerConfig{}
	flag.StringVar(&config.KubeConfig, "kubeconfig", "", "Path to kubeconfig file")
	flag.StringVar(&config.MetricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to")
	flag.BoolVar(&config.LeaderElection, "leader-elect", false, "Enable leader election for controller manager")
	flag.StringVar(&config.Region, "region", "us-east-1", "AWS region")
	flag.IntVar(&config.Workers, "workers", 1, "Number of worker goroutines")
	flag.Parse()

	klog.Infof("Starting Unified Bedrock Controller version %s", version)
	klog.Infof("Configuration: Region=%s, Workers=%d", config.Region, config.Workers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	setupSignalHandling(cancel)

	// Create Kubernetes client
	_, restConfig, err := createKubernetesClient(config.KubeConfig)
	if err != nil {
		klog.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Create AWS client
	bedrockClient, err := createBedrockClient(ctx, config.Region)
	if err != nil {
		klog.Fatalf("Failed to create Bedrock client: %v", err)
	}

	// Create event recorder
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	recorder := eventBroadcaster.NewRecorder(customScheme,
		corev1.EventSource{
			Component: "bedrock-controller",
		})

	// Create controller (which will create the reconciler internally)
	ctrl, err := createController(restConfig, bedrockClient, recorder, config.Workers)
	if err != nil {
		klog.Fatalf("Failed to create controller: %v", err)
	}

	klog.Info("Starting controller")
	if err := ctrl.Run(ctx); err != nil {
		klog.Fatalf("Controller failed: %v", err)
	}

	klog.Info("Controller stopped")
}

// setupSignalHandling sets up graceful shutdown on SIGTERM and SIGINT
func setupSignalHandling(cancel context.CancelFunc) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		klog.Info("Received shutdown signal, shutting down gracefully...")
		cancel()
		<-c
		klog.Info("Received second shutdown signal, shutting down immediately...")
		os.Exit(1)
	}()
}

// createKubernetesClient creates a Kubernetes client
func createKubernetesClient(kubeconfigPath string) (kubernetes.Interface, *rest.Config, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath != "" {
		// Use explicit kubeconfig path
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	} else {
		// Try in-cluster config first, then fall back to default kubeconfig
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig using loading rules
			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			configOverrides := &clientcmd.ConfigOverrides{}
			config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				loadingRules, configOverrides).ClientConfig()
		}
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return clientset, config, nil
}

// createBedrockClient creates an AWS Bedrock client
func createBedrockClient(ctx context.Context, region string) (controller.BedrockClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create both AWS SDK clients
	agentClient := bedrockagent.NewFromConfig(cfg)
	bedrockClient := bedrock.NewFromConfig(cfg)

	// Create our unified client wrapper
	return controller.NewBedrockClient(agentClient, bedrockClient, cfg), nil
}

// Controller represents our custom controller
type Controller struct {
	informer   cache.SharedIndexInformer
	queue      workqueue.RateLimitingInterface
	reconciler *controller.Reconciler
	workers    int
}

// createController creates a new controller instance
func createController(config *rest.Config, bedrockClient controller.BedrockClient, recorder record.EventRecorder, workers int) (*Controller, error) {
	// Create a REST client for our custom resource
	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{
		Group:   "bedrock.aws.example.com",
		Version: "v1",
	}
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(customScheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	restClient, err := rest.RESTClientFor(&crdConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	// Create reconciler
	reconciler := &controller.Reconciler{
		BedrockClient: bedrockClient,
		Scheme:        customScheme,
		Recorder:      recorder,
		RestClient:    restClient,
	}

	// Create list/watch client for BedrockResource
	listWatcher := cache.NewListWatchFromClient(
		restClient,
		"bedrockresources",
		metav1.NamespaceAll,
		fields.Everything(),
	)

	// Create informer
	informer := cache.NewSharedIndexInformer(
		listWatcher,
		&controller.BedrockResource{},
		time.Minute*10, // Resync period
		cache.Indexers{},
	)

	// Create work queue with rate limiting
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Add event handlers
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	})

	return &Controller{
		informer:   informer,
		queue:      queue,
		reconciler: reconciler,
		workers:    workers,
	}, nil
}

// Run starts the controller
func (c *Controller) Run(ctx context.Context) error {
	defer c.queue.ShutDown()

	klog.Info("Starting informer")
	go c.informer.Run(ctx.Done())

	// Wait for the caches to sync
	klog.Info("Waiting for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		return fmt.Errorf("failed to wait for cache sync")
	}

	klog.Infof("Starting %d workers", c.workers)
	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}

	<-ctx.Done()
	klog.Info("Stopping controller")
	return nil
}

// runWorker runs a worker goroutine
func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

// processNextWorkItem processes the next work item from the queue
func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.syncHandler(ctx, key.(string))
	if err == nil {
		c.queue.Forget(key)
		return true
	}

	klog.Errorf("Error syncing %q: %v", key, err)
	c.queue.AddRateLimited(key)
	return true
}

// syncHandler handles the synchronization of a single resource
func (c *Controller) syncHandler(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid resource key: %s", key)
	}

	// Get the resource from the informer cache
	obj, exists, err := c.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("failed to get resource from cache: %w", err)
	}

	if !exists {
		klog.Infof("Resource %s/%s no longer exists", namespace, name)
		return nil
	}

	resource, ok := obj.(*controller.BedrockResource)
	if !ok {
		return fmt.Errorf("expected BedrockResource, got %T", obj)
	}

	// Create a copy to avoid modifying the cached object
	resourceCopy := resource.DeepCopy()

	// Add context with resource information
	resourceCtx := klog.NewContext(ctx, klog.LoggerWithValues(klog.FromContext(ctx),
		"resource", resourceCopy.Name,
		"namespace", resourceCopy.Namespace,
		"type", resourceCopy.Spec.Type,
	))

	// Reconcile the resource
	result := c.reconciler.Reconcile(resourceCtx, resourceCopy)

	// Update resource status in Kubernetes
	if err := c.reconciler.UpdateResourceStatus(resourceCtx, resourceCopy); err != nil {
		klog.FromContext(ctx).Error(err, "Failed to update resource status")
		// Don't return the error - we'll retry on the next reconcile
	}

	// Handle requeue
	if result.Error != nil {
		return result.Error
	}

	if result.Requeue {
		if result.RequeueAfter > 0 {
			c.queue.AddAfter(key, result.RequeueAfter)
		} else {
			c.queue.AddRateLimited(key)
		}
	}

	return nil
}
