package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	controller "github.com/odnielgonzalez/bedrock-manager-controller/controller"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

// TestKnowledgeBaseInfiniteLoopReproduction tests the infinite loop issue
// that was previously seen with Inference Profiles, now applied to Knowledge Bases
func TestKnowledgeBaseInfiniteLoopReproduction(t *testing.T) {
	// This test simulates the real controller pattern where:
	// 1. Each reconciliation gets a fresh copy from cache
	// 2. Status updates happen after reconciliation
	// 3. The next reconciliation should see the updated status
	// 4. Knowledge Bases should NOT be created multiple times

	mockClient := NewMockBedrockClient()
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Recorder:      recorder,
		// RestClient is nil to simulate unit test environment
	}

	// Simulate the original resource from cache (fresh each time)
	createFreshKBResource := func() *controller.BedrockResource {
		return &controller.BedrockResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-kb-infinite-loop",
				Namespace:       "default",
				ResourceVersion: "1000", // Simulate K8s resource version
				Generation:      1,
				Finalizers:      []string{controller.BedrockResourceFinalizer},
			},
			Spec: controller.BedrockResourceSpec{
				Type: "knowledgeBase",
				KnowledgeBase: &controller.KnowledgeBaseSpec{
					Name:              "TestKnowledgeBase",
					Description:       aws.String("Test for infinite loop prevention"),
					RoleArn:           "arn:aws:iam::123456789012:role/TestRole",
					EmbeddingModelArn: "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1",
					VectorStoreType:   "OPENSEARCH_SERVERLESS",
					OpenSearchServerlessConfiguration: &controller.OpenSearchServerlessConfig{
						CollectionArn:   "arn:aws:aoss:us-east-1:123456789012:collection/test-collection",
						VectorIndexName: "test-index",
						VectorField:     "vector",
						TextField:       "text",
						MetadataField:   "metadata",
					},
					Tags: map[string]string{
						"Environment": "test",
						"Purpose":     "infinite-loop-test",
					},
				},
			},
		}
	}

	ctx := context.Background()
	var savedStatus controller.BedrockResourceStatus

	// CYCLE 1: First reconciliation (should create knowledge base)
	t.Run("Cycle1_CreateKnowledgeBase", func(t *testing.T) {
		resource1 := createFreshKBResource()

		result1 := reconciler.Reconcile(ctx, resource1)
		require.NoError(t, result1.Error)

		// Verify knowledge base was created
		require.True(t, mockClient.CreateKnowledgeBaseCalled)
		require.Equal(t, 1, len(mockClient.GetKnowledgeBases()))
		require.NotNil(t, resource1.Status.KnowledgeBaseId)
		require.Equal(t, controller.PhasePrepared, resource1.Status.Phase)

		// Save the status (simulates what UpdateResourceStatus would persist)
		savedStatus = resource1.Status

		t.Logf("Created knowledge base with ID: %s", *resource1.Status.KnowledgeBaseId)
	})

	// Reset call tracking but keep the created knowledge base in mock state
	mockClient.CreateKnowledgeBaseCalled = false

	// CYCLE 2: Second reconciliation (should NOT create another knowledge base)
	// This simulates getting a fresh resource from cache with persisted status
	t.Run("Cycle2_ShouldNotCreateAnother", func(t *testing.T) {
		resource2 := createFreshKBResource()
		// Simulate that Kubernetes persisted the status from cycle 1
		resource2.Status = savedStatus

		result2 := reconciler.Reconcile(ctx, resource2)
		require.NoError(t, result2.Error)

		// Should NOT create another knowledge base
		require.False(t, mockClient.CreateKnowledgeBaseCalled,
			"Second reconciliation should not create another knowledge base when status has KnowledgeBaseId")
		require.Equal(t, 1, len(mockClient.GetKnowledgeBases()),
			"Should still have exactly 1 knowledge base")
		require.Equal(t, controller.PhasePrepared, resource2.Status.Phase)

		t.Logf("Second cycle correctly reused existing knowledge base: %s", *resource2.Status.KnowledgeBaseId)
	})

	// CYCLE 3: Third reconciliation (final verification)
	t.Run("Cycle3_StillStable", func(t *testing.T) {
		resource3 := createFreshKBResource()
		resource3.Status = savedStatus

		mockClient.CreateKnowledgeBaseCalled = false // Reset again

		result3 := reconciler.Reconcile(ctx, resource3)
		require.NoError(t, result3.Error)

		// Still should not create another knowledge base
		require.False(t, mockClient.CreateKnowledgeBaseCalled)
		require.Equal(t, 1, len(mockClient.GetKnowledgeBases()))
		require.Equal(t, controller.PhasePrepared, resource3.Status.Phase)

		t.Logf("Third cycle still stable with knowledge base: %s", *resource3.Status.KnowledgeBaseId)
	})
}

// TestKnowledgeBaseFinalizerStuckReproduction tests the finalizer stuck issue
// that was previously seen with Inference Profiles, now applied to Knowledge Bases
func TestKnowledgeBaseFinalizerStuckReproduction(t *testing.T) {
	mockClient := NewMockBedrockClient()
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Recorder:      recorder,
		// RestClient is nil - this was part of the original problem!
	}

	// Create and set up a resource with an existing knowledge base
	kbId := "test-finalizer-kb-123"
	kbArn := "arn:aws:bedrock:us-east-1:123456789012:knowledge-base/" + kbId

	// Add knowledge base to mock client state
	now := time.Now().UTC()
	existingKB := &types.KnowledgeBase{
		KnowledgeBaseId:  &kbId,
		KnowledgeBaseArn: &kbArn,
		Name:             aws.String("TestKnowledgeBase"),
		Status:           types.KnowledgeBaseStatusActive,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	mockClient.AddKnowledgeBase(existingKB)

	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-kb-finalizer",
			Namespace:         "default",
			ResourceVersion:   "2000",
			Generation:        1,
			DeletionTimestamp: &metav1.Time{Time: time.Now()}, // Marked for deletion
			Finalizers:        []string{controller.BedrockResourceFinalizer},
		},
		Spec: controller.BedrockResourceSpec{
			Type: "knowledgeBase",
			KnowledgeBase: &controller.KnowledgeBaseSpec{
				Name:              "TestKnowledgeBase",
				Description:       aws.String("Test for finalizer"),
				RoleArn:           "arn:aws:iam::123456789012:role/TestRole",
				EmbeddingModelArn: "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1",
				VectorStoreType:   "OPENSEARCH_SERVERLESS",
				OpenSearchServerlessConfiguration: &controller.OpenSearchServerlessConfig{
					CollectionArn:   "arn:aws:aoss:us-east-1:123456789012:collection/test-collection",
					VectorIndexName: "test-index",
					VectorField:     "vector",
					TextField:       "text",
					MetadataField:   "metadata",
				},
			},
		},
		Status: controller.BedrockResourceStatus{
			KnowledgeBaseId:  &kbId,
			KnowledgeBaseArn: &kbArn,
			ResourceId:       &kbId,
			ResourceArn:      &kbArn,
			Phase:            controller.PhasePrepared,
		},
	}

	ctx := context.Background()

	t.Run("DeletionShouldSucceed", func(t *testing.T) {
		// Verify knowledge base exists before deletion
		require.Equal(t, 1, len(mockClient.GetKnowledgeBases()))

		result := reconciler.Reconcile(ctx, resource)

		// Should not error
		require.NoError(t, result.Error)

		// Should delete the AWS knowledge base
		require.True(t, mockClient.DeleteKnowledgeBaseCalled)

		// Knowledge base should be removed from mock state
		require.Equal(t, 0, len(mockClient.GetKnowledgeBases()))

		// Finalizer should be removed from the resource
		require.NotContains(t, resource.Finalizers, controller.BedrockResourceFinalizer,
			"Finalizer should be removed after successful deletion")

		t.Logf("Deletion completed successfully, finalizer removed")
	})
}

// TestKnowledgeBaseCacheStaleStatusIssue tests the case where the controller cache
// gives stale status information leading to infinite loops for Knowledge Bases
func TestKnowledgeBaseCacheStaleStatusIssue(t *testing.T) {
	mockClient := NewMockBedrockClient()
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Recorder:      recorder,
	}

	// Simulate the problematic scenario where:
	// 1. Controller creates a knowledge base and updates local status
	// 2. Status update to K8s "succeeds" but status isn't actually persisted
	// 3. Next reconciliation gets stale data from cache (no KnowledgeBaseId)
	// 4. Controller should now discover the existing knowledge base by name instead of creating another

	kbName := "TestStaleCache"
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-kb-stale-cache",
			Namespace:       "default",
			ResourceVersion: "3000",
			Generation:      1,
			Finalizers:      []string{controller.BedrockResourceFinalizer},
		},
		Spec: controller.BedrockResourceSpec{
			Type: "knowledgeBase",
			KnowledgeBase: &controller.KnowledgeBaseSpec{
				Name:              kbName,
				Description:       aws.String("Test for stale cache"),
				RoleArn:           "arn:aws:iam::123456789012:role/TestRole",
				EmbeddingModelArn: "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1",
				VectorStoreType:   "OPENSEARCH_SERVERLESS",
				OpenSearchServerlessConfiguration: &controller.OpenSearchServerlessConfig{
					CollectionArn:   "arn:aws:aoss:us-east-1:123456789012:collection/test-collection",
					VectorIndexName: "test-index",
					VectorField:     "vector",
					TextField:       "text",
					MetadataField:   "metadata",
				},
			},
		},
		// Status is empty (simulating fresh resource from cache)
	}

	ctx := context.Background()

	// First reconciliation - creates knowledge base
	result1 := reconciler.Reconcile(ctx, resource)
	require.NoError(t, result1.Error)
	require.True(t, mockClient.CreateKnowledgeBaseCalled)
	require.NotNil(t, resource.Status.KnowledgeBaseId)

	firstKBId := *resource.Status.KnowledgeBaseId
	t.Logf("First reconciliation created knowledge base: %s", firstKBId)

	// Simulate that the knowledge base exists in AWS but the K8s status wasn't persisted
	// (due to optimistic concurrency conflicts or other issues)
	// This means the next reconciliation will see empty status
	resourceWithStaleStatus := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-kb-stale-cache",
			Namespace:       "default",
			ResourceVersion: "3001", // Bumped resource version
			Generation:      1,
			Finalizers:      []string{controller.BedrockResourceFinalizer},
		},
		Spec: controller.BedrockResourceSpec{
			Type: "knowledgeBase",
			KnowledgeBase: &controller.KnowledgeBaseSpec{
				Name:              kbName, // Same name as first reconciliation
				Description:       aws.String("Test for stale cache"),
				RoleArn:           "arn:aws:iam::123456789012:role/TestRole",
				EmbeddingModelArn: "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1",
				VectorStoreType:   "OPENSEARCH_SERVERLESS",
				OpenSearchServerlessConfiguration: &controller.OpenSearchServerlessConfig{
					CollectionArn:   "arn:aws:aoss:us-east-1:123456789012:collection/test-collection",
					VectorIndexName: "test-index",
					VectorField:     "vector",
					TextField:       "text",
					MetadataField:   "metadata",
				},
			},
		},
		// Status is EMPTY again! This simulates the cache serving stale data
		Status: controller.BedrockResourceStatus{},
	}

	// Reset call tracking but keep the created knowledge base in mock state
	mockClient.CreateKnowledgeBaseCalled = false

	// Second reconciliation with stale cache - this is where the bug could occur
	result2 := reconciler.Reconcile(ctx, resourceWithStaleStatus)
	require.NoError(t, result2.Error)

	// THE BUG CHECK: This should NOT create another knowledge base
	// The controller should discover the existing one by name
	if mockClient.CreateKnowledgeBaseCalled {
		t.Errorf("BUG REPRODUCED: Second reconciliation created another knowledge base when status was stale!")
		t.Errorf("Knowledge base count: %d (should be 1)", len(mockClient.GetKnowledgeBases()))

		// This reproduces the infinite loop issue
		kbs := mockClient.GetKnowledgeBases()
		if len(kbs) > 1 {
			for id, kb := range kbs {
				t.Logf("Knowledge Base %s: %+v", id, kb)
			}
		}
	} else {
		t.Logf("Good: Second reconciliation did not create another knowledge base")
		require.Equal(t, 1, len(mockClient.GetKnowledgeBases()))

		// Verify that the controller found the existing KB and updated status
		require.NotNil(t, resourceWithStaleStatus.Status.KnowledgeBaseId)
		require.Equal(t, firstKBId, *resourceWithStaleStatus.Status.KnowledgeBaseId)
		require.True(t, mockClient.ListKnowledgeBasesCalled, "Should have called ListKnowledgeBases to find existing KB")
	}
}

// TestKnowledgeBaseOptimisticConcurrencyHandling tests the optimistic concurrency
// conflict resolution that was a major issue in previous phases
func TestKnowledgeBaseOptimisticConcurrencyHandling(t *testing.T) {
	mockClient := NewMockBedrockClient()
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Recorder:      recorder,
	}

	// Simulate multiple workers trying to reconcile the same resource
	// This can happen in real Kubernetes controllers due to:
	// 1. Multiple controller replicas
	// 2. Quick re-queuing of work items
	// 3. Cache inconsistencies

	createResourceTemplate := func(resourceVersion string) *controller.BedrockResource {
		return &controller.BedrockResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-kb-concurrency",
				Namespace:       "default",
				ResourceVersion: resourceVersion,
				Generation:      1,
				Finalizers:      []string{controller.BedrockResourceFinalizer},
			},
			Spec: controller.BedrockResourceSpec{
				Type: "knowledgeBase",
				KnowledgeBase: &controller.KnowledgeBaseSpec{
					Name:              "ConcurrencyTestKB",
					Description:       aws.String("Test for concurrency handling"),
					RoleArn:           "arn:aws:iam::123456789012:role/TestRole",
					EmbeddingModelArn: "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1",
					VectorStoreType:   "OPENSEARCH_SERVERLESS",
					OpenSearchServerlessConfiguration: &controller.OpenSearchServerlessConfig{
						CollectionArn:   "arn:aws:aoss:us-east-1:123456789012:collection/test-collection",
						VectorIndexName: "test-index",
						VectorField:     "vector",
						TextField:       "text",
						MetadataField:   "metadata",
					},
				},
			},
		}
	}

	ctx := context.Background()

	t.Run("FirstWorkerCreatesKB", func(t *testing.T) {
		worker1Resource := createResourceTemplate("1000")

		result := reconciler.Reconcile(ctx, worker1Resource)
		require.NoError(t, result.Error)

		// First worker should create the knowledge base
		require.True(t, mockClient.CreateKnowledgeBaseCalled)
		require.Equal(t, 1, len(mockClient.GetKnowledgeBases()))
		require.NotNil(t, worker1Resource.Status.KnowledgeBaseId)

		t.Logf("First worker created KB: %s", *worker1Resource.Status.KnowledgeBaseId)
	})

	// Reset call tracking but keep the created KB in mock state
	mockClient.CreateKnowledgeBaseCalled = false

	t.Run("SecondWorkerWithOlderCacheDoesntDuplicate", func(t *testing.T) {
		// Simulate a second worker with an older cached version (no status)
		worker2Resource := createResourceTemplate("999") // Older resource version
		// Status is empty, simulating stale cache

		result := reconciler.Reconcile(ctx, worker2Resource)
		require.NoError(t, result.Error)

		// Second worker should NOT create another knowledge base
		// It should discover the existing one through name-based lookup
		require.False(t, mockClient.CreateKnowledgeBaseCalled,
			"Second worker should not create another KB when one already exists")
		require.Equal(t, 1, len(mockClient.GetKnowledgeBases()))

		// Should have discovered and updated status
		require.NotNil(t, worker2Resource.Status.KnowledgeBaseId)
		require.True(t, mockClient.ListKnowledgeBasesCalled,
			"Should have called ListKnowledgeBases to discover existing KB")

		t.Logf("Second worker found existing KB: %s", *worker2Resource.Status.KnowledgeBaseId)
	})
}

// TestKnowledgeBaseDeletionRaceCondition tests race conditions during deletion
// that could lead to stuck finalizers or partial cleanups
func TestKnowledgeBaseDeletionRaceCondition(t *testing.T) {
	t.Run("DeleteNonExistentKB", func(t *testing.T) {
		mockClient := NewMockBedrockClient()
		recorder := record.NewFakeRecorder(10)
		reconciler := &controller.Reconciler{
			BedrockClient: mockClient,
			Recorder:      recorder,
		}

		// Resource that thinks it has a KB but AWS doesn't have it
		// This can happen if:
		// 1. KB was manually deleted from AWS
		// 2. Previous deletion attempt partially succeeded
		// 3. Multiple deletion attempts
		resource := &controller.BedrockResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-kb-deletion-race",
				Namespace:         "default",
				ResourceVersion:   "4000",
				Generation:        1,
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
				Finalizers:        []string{controller.BedrockResourceFinalizer},
			},
			Spec: controller.BedrockResourceSpec{
				Type: "knowledgeBase",
				KnowledgeBase: &controller.KnowledgeBaseSpec{
					Name:    "NonExistentKB",
					RoleArn: "arn:aws:iam::123456789012:role/TestRole",
				},
			},
			Status: controller.BedrockResourceStatus{
				KnowledgeBaseId: aws.String("non-existent-kb-123"),
				Phase:           controller.PhasePrepared,
			},
		}

		ctx := context.Background()
		result := reconciler.Reconcile(ctx, resource)

		// Should not error even when KB doesn't exist
		require.NoError(t, result.Error)

		// Should attempt to delete (gracefully handle not found)
		require.True(t, mockClient.DeleteKnowledgeBaseCalled)

		// Finalizer should be removed even though KB wasn't found
		require.NotContains(t, resource.Finalizers, controller.BedrockResourceFinalizer,
			"Finalizer should be removed even when KB doesn't exist in AWS")

		t.Logf("Deletion of non-existent KB handled gracefully")
	})

	t.Run("DeleteWithAWSError", func(t *testing.T) {
		mockClient := NewMockBedrockClient()
		mockClient.SetDeleteKnowledgeBaseError(errors.New("AccessDeniedException: Insufficient permissions"))

		recorder := record.NewFakeRecorder(10)
		reconciler := &controller.Reconciler{
			BedrockClient: mockClient,
			Recorder:      recorder,
		}

		// Add a KB to mock state
		kbId := "kb-with-error-123"
		existingKB := &types.KnowledgeBase{
			KnowledgeBaseId:  &kbId,
			KnowledgeBaseArn: aws.String("arn:aws:bedrock:us-east-1:123456789012:knowledge-base/" + kbId),
			Name:             aws.String("ErrorKB"),
			Status:           types.KnowledgeBaseStatusActive,
		}
		mockClient.AddKnowledgeBase(existingKB)

		resource := &controller.BedrockResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-kb-deletion-error",
				Namespace:         "default",
				ResourceVersion:   "4001",
				Generation:        1,
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
				Finalizers:        []string{controller.BedrockResourceFinalizer},
			},
			Spec: controller.BedrockResourceSpec{
				Type: "knowledgeBase",
				KnowledgeBase: &controller.KnowledgeBaseSpec{
					Name:    "ErrorKB",
					RoleArn: "arn:aws:iam::123456789012:role/TestRole",
				},
			},
			Status: controller.BedrockResourceStatus{
				KnowledgeBaseId: &kbId,
				Phase:           controller.PhasePrepared,
			},
		}

		ctx := context.Background()
		result := reconciler.Reconcile(ctx, resource)

		// Should error and requeue
		require.Error(t, result.Error)
		require.Equal(t, 30*time.Second, result.RequeueAfter)

		// Should attempt to delete
		require.True(t, mockClient.DeleteKnowledgeBaseCalled)

		// Finalizer should still be present due to error
		require.Contains(t, resource.Finalizers, controller.BedrockResourceFinalizer,
			"Finalizer should remain when deletion fails")

		t.Logf("Deletion error handled correctly with requeue")
	})
}

// TestKnowledgeBaseResourceVersionConflicts tests handling of resource version conflicts
// that can occur during status updates in high-traffic scenarios
func TestKnowledgeBaseResourceVersionConflicts(t *testing.T) {
	mockClient := NewMockBedrockClient()
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Recorder:      recorder,
	}

	// Simulate the scenario where:
	// 1. Worker A reads resource with version 1000
	// 2. Worker B reads resource with version 1000
	// 3. Worker A updates resource → version becomes 1001
	// 4. Worker B tries to update → conflict (expects 1000, but current is 1001)

	baseResource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-kb-version-conflict",
			Namespace:       "default",
			ResourceVersion: "1000",
			Generation:      1,
			Finalizers:      []string{controller.BedrockResourceFinalizer},
		},
		Spec: controller.BedrockResourceSpec{
			Type: "knowledgeBase",
			KnowledgeBase: &controller.KnowledgeBaseSpec{
				Name:              "VersionConflictKB",
				Description:       aws.String("Test for version conflicts"),
				RoleArn:           "arn:aws:iam::123456789012:role/TestRole",
				EmbeddingModelArn: "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1",
				VectorStoreType:   "OPENSEARCH_SERVERLESS",
				OpenSearchServerlessConfiguration: &controller.OpenSearchServerlessConfig{
					CollectionArn:   "arn:aws:aoss:us-east-1:123456789012:collection/test-collection",
					VectorIndexName: "test-index",
					VectorField:     "vector",
					TextField:       "text",
					MetadataField:   "metadata",
				},
			},
		},
	}

	ctx := context.Background()

	t.Run("FirstWorkerSucceeds", func(t *testing.T) {
		workerA := &controller.BedrockResource{}
		*workerA = *baseResource // Copy

		result := reconciler.Reconcile(ctx, workerA)
		require.NoError(t, result.Error)

		// Should create KB successfully
		require.True(t, mockClient.CreateKnowledgeBaseCalled)
		require.NotNil(t, workerA.Status.KnowledgeBaseId)
		require.Equal(t, controller.PhasePrepared, workerA.Status.Phase)

		t.Logf("Worker A created KB: %s", *workerA.Status.KnowledgeBaseId)
	})

	// Reset tracking but keep the created KB
	mockClient.CreateKnowledgeBaseCalled = false

	t.Run("SecondWorkerWithStaleResourceVersion", func(t *testing.T) {
		// Worker B has the same old resource version but KB already exists
		workerB := &controller.BedrockResource{}
		*workerB = *baseResource // Copy (same resource version 1000)

		result := reconciler.Reconcile(ctx, workerB)
		require.NoError(t, result.Error)

		// Should NOT create another KB (should discover existing one)
		require.False(t, mockClient.CreateKnowledgeBaseCalled,
			"Worker B should not create another KB")
		require.Equal(t, 1, len(mockClient.GetKnowledgeBases()))

		// Should have discovered existing KB and updated status
		require.NotNil(t, workerB.Status.KnowledgeBaseId)
		require.Equal(t, controller.PhasePrepared, workerB.Status.Phase)

		t.Logf("Worker B found existing KB: %s", *workerB.Status.KnowledgeBaseId)
	})
}
