package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	"github.com/odnielgonzalez/bedrock-manager-controller/controller"
)

// Helper function to create a properly configured reconciler for testing
func createTestReconciler(mockClient *MockBedrockClient) *controller.Reconciler {
	return &controller.Reconciler{
		BedrockClient: mockClient,
		Scheme:        runtime.NewScheme(),
		Recorder:      record.NewFakeRecorder(10),
	}
}

func TestKnowledgeBaseCreationSuccess(t *testing.T) {
	// Arrange
	mockClient := NewMockBedrockClient()
	reconciler := createTestReconciler(mockClient)

	ctx := context.Background()
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-kb",
			Namespace: "default",
		},
		Spec: controller.BedrockResourceSpec{
			Type: "knowledgeBase",
			KnowledgeBase: &controller.KnowledgeBaseSpec{
				Name:              "TestKnowledgeBase",
				Description:       aws.String("Test knowledge base for unit testing"),
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
					"Purpose":     "unit-testing",
				},
			},
		},
	}

	// Act
	result := reconciler.Reconcile(ctx, resource)

	// Assert
	require.NoError(t, result.Error)
	require.False(t, result.Requeue)
	require.Equal(t, time.Duration(0), result.RequeueAfter)

	// Verify the knowledge base was created
	require.True(t, mockClient.CreateKnowledgeBaseCalled)
	require.NotNil(t, mockClient.CreateKnowledgeBaseInput)
	require.Equal(t, "TestKnowledgeBase", *mockClient.CreateKnowledgeBaseInput.Name)
	require.Equal(t, "Test knowledge base for unit testing", *mockClient.CreateKnowledgeBaseInput.Description)
	require.Equal(t, "arn:aws:iam::123456789012:role/TestRole", *mockClient.CreateKnowledgeBaseInput.RoleArn)

	// Verify status was updated
	require.NotNil(t, resource.Status.KnowledgeBaseId)
	require.NotNil(t, resource.Status.KnowledgeBaseArn)
	require.Equal(t, controller.PhasePrepared, resource.Status.Phase)

	// Verify the knowledge base exists in mock state
	kbs := mockClient.GetKnowledgeBases()
	require.Len(t, kbs, 1)
}

func TestKnowledgeBaseCreationWithAWSError(t *testing.T) {
	// Arrange
	mockClient := NewMockBedrockClient()
	mockClient.SetCreateKnowledgeBaseError(errors.New("AWS API Error"))

	reconciler := createTestReconciler(mockClient)

	ctx := context.Background()
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-kb-error",
			Namespace: "default",
		},
		Spec: controller.BedrockResourceSpec{
			Type: "knowledgeBase",
			KnowledgeBase: &controller.KnowledgeBaseSpec{
				Name:              "TestKnowledgeBase",
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

	// Act
	result := reconciler.Reconcile(ctx, resource)

	// Assert
	require.Error(t, result.Error)
	require.Equal(t, time.Minute, result.RequeueAfter)
	require.True(t, mockClient.CreateKnowledgeBaseCalled)

	// Verify status indicates error
	require.Equal(t, controller.PhaseFailed, resource.Status.Phase)
}

func TestKnowledgeBaseMissingSpec(t *testing.T) {
	// Arrange
	mockClient := NewMockBedrockClient()
	reconciler := createTestReconciler(mockClient)

	ctx := context.Background()
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-kb-no-spec",
			Namespace: "default",
		},
		Spec: controller.BedrockResourceSpec{
			Type: "knowledgeBase",
			// Missing KnowledgeBase spec
		},
	}

	// Act
	result := reconciler.Reconcile(ctx, resource)

	// Assert
	require.Error(t, result.Error)
	require.Contains(t, result.Error.Error(), "knowledgeBase spec is required")
	require.False(t, mockClient.CreateKnowledgeBaseCalled)
}

func TestKnowledgeBaseReconcileExisting(t *testing.T) {
	// Arrange
	mockClient := NewMockBedrockClient()

	// Pre-populate with existing knowledge base
	kbId := "existing-kb-123"
	kbArn := "arn:aws:bedrock:us-east-1:123456789012:knowledge-base/" + kbId
	now := time.Now().UTC()
	existingKB := &types.KnowledgeBase{
		KnowledgeBaseId:  &kbId,
		KnowledgeBaseArn: &kbArn,
		Name:             aws.String("TestKnowledgeBase"),
		Description:      aws.String("Existing knowledge base"),
		RoleArn:          aws.String("arn:aws:iam::123456789012:role/TestRole"),
		Status:           types.KnowledgeBaseStatusActive,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	mockClient.AddKnowledgeBase(existingKB)

	reconciler := createTestReconciler(mockClient)

	ctx := context.Background()
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-kb-existing",
			Namespace: "default",
		},
		Spec: controller.BedrockResourceSpec{
			Type: "knowledgeBase",
			KnowledgeBase: &controller.KnowledgeBaseSpec{
				Name:              "TestKnowledgeBase",
				Description:       aws.String("Updated description"),
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
			KnowledgeBaseId: &kbId, // Pre-existing status
		},
	}

	// Act
	result := reconciler.Reconcile(ctx, resource)

	// Assert
	require.NoError(t, result.Error)
	require.True(t, mockClient.GetKnowledgeBaseCalled)
	require.True(t, mockClient.UpdateKnowledgeBaseCalled)  // Should update because description changed
	require.False(t, mockClient.CreateKnowledgeBaseCalled) // Should NOT create new one

	// Verify status was updated from existing knowledge base
	require.Equal(t, kbId, *resource.Status.KnowledgeBaseId)
	require.Equal(t, kbArn, *resource.Status.KnowledgeBaseArn)
	require.Equal(t, controller.PhasePrepared, resource.Status.Phase)
}

func TestKnowledgeBaseDeletion(t *testing.T) {
	// Arrange
	mockClient := NewMockBedrockClient()

	// Pre-populate with existing knowledge base
	kbId := "kb-to-delete-123"
	existingKB := &types.KnowledgeBase{
		KnowledgeBaseId:  &kbId,
		KnowledgeBaseArn: aws.String("arn:aws:bedrock:us-east-1:123456789012:knowledge-base/" + kbId),
		Name:             aws.String("TestKnowledgeBase"),
		Status:           types.KnowledgeBaseStatusActive,
	}
	mockClient.AddKnowledgeBase(existingKB)

	reconciler := createTestReconciler(mockClient)

	ctx := context.Background()
	now := metav1.NewTime(time.Now())
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-kb-delete",
			Namespace:         "default",
			DeletionTimestamp: &now, // Mark for deletion
			Finalizers:        []string{controller.BedrockResourceFinalizer},
		},
		Spec: controller.BedrockResourceSpec{
			Type: "knowledgeBase",
			KnowledgeBase: &controller.KnowledgeBaseSpec{
				Name:              "TestKnowledgeBase",
				RoleArn:           "arn:aws:iam::123456789012:role/TestRole",
				EmbeddingModelArn: "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1",
				VectorStoreType:   "OPENSEARCH_SERVERLESS",
			},
		},
		Status: controller.BedrockResourceStatus{
			KnowledgeBaseId: &kbId,
		},
	}

	// Act
	result := reconciler.Reconcile(ctx, resource)

	// Assert
	require.NoError(t, result.Error)
	require.True(t, mockClient.DeleteKnowledgeBaseCalled)
	require.Equal(t, kbId, *mockClient.DeleteKnowledgeBaseInput.KnowledgeBaseId)

	// Verify knowledge base was removed from mock state
	kbs := mockClient.GetKnowledgeBases()
	require.Empty(t, kbs)

	// Verify finalizer was removed
	require.Empty(t, resource.Finalizers)
}

func TestKnowledgeBaseDeletionWhenNotFound(t *testing.T) {
	// Arrange
	mockClient := NewMockBedrockClient()
	reconciler := createTestReconciler(mockClient)

	ctx := context.Background()
	now := metav1.NewTime(time.Now())
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-kb-delete-404",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{controller.BedrockResourceFinalizer},
		},
		Spec: controller.BedrockResourceSpec{
			Type: "knowledgeBase",
			KnowledgeBase: &controller.KnowledgeBaseSpec{
				Name:    "TestKnowledgeBase",
				RoleArn: "arn:aws:iam::123456789012:role/TestRole",
			},
		},
		Status: controller.BedrockResourceStatus{
			KnowledgeBaseId: aws.String("non-existent-kb"),
		},
	}

	// Act
	result := reconciler.Reconcile(ctx, resource)

	// Assert
	require.NoError(t, result.Error) // Should succeed even when KB not found
	require.True(t, mockClient.DeleteKnowledgeBaseCalled)

	// Verify finalizer was removed despite 404
	require.Empty(t, resource.Finalizers)
}

func TestKnowledgeBaseDeletionWithAWSError(t *testing.T) {
	// Arrange
	mockClient := NewMockBedrockClient()
	mockClient.SetDeleteKnowledgeBaseError(errors.New("AWS Deletion Error"))

	reconciler := createTestReconciler(mockClient)

	ctx := context.Background()
	now := metav1.NewTime(time.Now())
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-kb-delete-error",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{controller.BedrockResourceFinalizer},
		},
		Spec: controller.BedrockResourceSpec{
			Type: "knowledgeBase",
			KnowledgeBase: &controller.KnowledgeBaseSpec{
				Name:    "TestKnowledgeBase",
				RoleArn: "arn:aws:iam::123456789012:role/TestRole",
			},
		},
		Status: controller.BedrockResourceStatus{
			KnowledgeBaseId: aws.String("kb-with-error"),
		},
	}

	// Act
	result := reconciler.Reconcile(ctx, resource)

	// Assert
	require.Error(t, result.Error)
	require.Equal(t, 30*time.Second, result.RequeueAfter)
	require.True(t, mockClient.DeleteKnowledgeBaseCalled)

	// Finalizer should still be present due to error
	require.Contains(t, resource.Finalizers, controller.BedrockResourceFinalizer)
}

func TestKnowledgeBaseConversionHelpers(t *testing.T) {
	t.Run("convertKnowledgeBaseSpecToCreateInput", func(t *testing.T) {
		// Arrange
		spec := &controller.KnowledgeBaseSpec{
			Name:              "TestKnowledgeBase",
			Description:       aws.String("Test description"),
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
				"Team":        "platform",
			},
		}

		// Act
		input := controller.ConvertKnowledgeBaseSpecToCreateInput(spec)

		// Assert
		require.NotNil(t, input)
		require.Equal(t, "TestKnowledgeBase", *input.Name)
		require.Equal(t, "Test description", *input.Description)
		require.Equal(t, "arn:aws:iam::123456789012:role/TestRole", *input.RoleArn)

		// Verify knowledge base configuration
		require.NotNil(t, input.KnowledgeBaseConfiguration)
		require.Equal(t, types.KnowledgeBaseTypeVector, input.KnowledgeBaseConfiguration.Type)
		require.NotNil(t, input.KnowledgeBaseConfiguration.VectorKnowledgeBaseConfiguration)
		require.Equal(t, "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1",
			*input.KnowledgeBaseConfiguration.VectorKnowledgeBaseConfiguration.EmbeddingModelArn)

		// Verify storage configuration
		require.NotNil(t, input.StorageConfiguration)
		require.Equal(t, types.KnowledgeBaseStorageTypeOpensearchServerless, input.StorageConfiguration.Type)
		require.NotNil(t, input.StorageConfiguration.OpensearchServerlessConfiguration)

		// Verify tags
		require.Equal(t, "test", input.Tags["Environment"])
		require.Equal(t, "platform", input.Tags["Team"])
	})

	t.Run("convertAWSKnowledgeBaseToStatus", func(t *testing.T) {
		// Arrange
		kbId := "test-kb-123"
		kbArn := "arn:aws:bedrock:us-east-1:123456789012:knowledge-base/" + kbId
		kb := &types.KnowledgeBase{
			KnowledgeBaseId:  &kbId,
			KnowledgeBaseArn: &kbArn,
			Name:             aws.String("TestKnowledgeBase"),
			Status:           types.KnowledgeBaseStatusActive,
			FailureReasons:   []string{"test-failure-reason"},
		}

		// Act
		status := controller.ConvertAWSKnowledgeBaseToStatus(kb)

		// Assert
		require.NotNil(t, status)
		require.Equal(t, controller.PhasePrepared, status.Phase) // Active maps to Prepared
		require.Equal(t, kbId, *status.KnowledgeBaseId)
		require.Equal(t, kbArn, *status.KnowledgeBaseArn)
		require.Equal(t, kbId, *status.ResourceId)
		require.Equal(t, kbArn, *status.ResourceArn)
		require.Equal(t, "ACTIVE", *status.KnowledgeBaseStatus)
		require.Equal(t, []string{"test-failure-reason"}, status.FailureReasons)
	})

	t.Run("mapKnowledgeBaseStatusToPhase", func(t *testing.T) {
		testCases := []struct {
			awsStatus     types.KnowledgeBaseStatus
			expectedPhase string
		}{
			{types.KnowledgeBaseStatusActive, controller.PhasePrepared},
			{types.KnowledgeBaseStatusCreating, controller.PhaseCreating},
			{types.KnowledgeBaseStatusUpdating, controller.PhaseUpdating},
			{types.KnowledgeBaseStatusDeleting, controller.PhaseDeleting},
			{types.KnowledgeBaseStatusFailed, controller.PhaseFailed},
		}

		for _, tc := range testCases {
			t.Run(string(tc.awsStatus), func(t *testing.T) {
				phase := controller.MapKnowledgeBaseStatusToPhase(tc.awsStatus)
				require.Equal(t, tc.expectedPhase, phase)
			})
		}
	})
}

func TestKnowledgeBaseStaleCache(t *testing.T) {
	// This test simulates the stale cache scenario where the controller
	// receives a resource with empty status even though a KB exists in AWS

	// Arrange
	mockClient := NewMockBedrockClient()

	// Pre-populate AWS with existing knowledge base
	kbId := "existing-kb-456"
	kbArn := "arn:aws:bedrock:us-east-1:123456789012:knowledge-base/" + kbId
	now := time.Now().UTC()
	existingKB := &types.KnowledgeBase{
		KnowledgeBaseId:  &kbId,
		KnowledgeBaseArn: &kbArn,
		Name:             aws.String("MyKnowledgeBase"),
		Status:           types.KnowledgeBaseStatusActive,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	mockClient.AddKnowledgeBase(existingKB)

	reconciler := createTestReconciler(mockClient)

	ctx := context.Background()
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-kb-stale-cache",
			Namespace: "default",
		},
		Spec: controller.BedrockResourceSpec{
			Type: "knowledgeBase",
			KnowledgeBase: &controller.KnowledgeBaseSpec{
				Name:              "MyKnowledgeBase", // Same name as existing
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
		// Empty status simulates stale cache
		Status: controller.BedrockResourceStatus{},
	}

	// Act
	result := reconciler.Reconcile(ctx, resource)

	// Assert
	require.NoError(t, result.Error)

	// Should discover existing KB by name, not create a new one
	require.True(t, mockClient.ListKnowledgeBasesCalled)
	require.False(t, mockClient.CreateKnowledgeBaseCalled)

	// Status should be updated with discovered KB
	require.Equal(t, kbId, *resource.Status.KnowledgeBaseId)
	require.Equal(t, kbArn, *resource.Status.KnowledgeBaseArn)
	require.Equal(t, controller.PhasePrepared, resource.Status.Phase)

	// Should still have only one KB in mock (no duplicates)
	kbs := mockClient.GetKnowledgeBases()
	require.Len(t, kbs, 1)
}

// Make helper functions public for testing
func init() {
	// This allows the test to access private functions by making them public
	// In a real implementation, these would be methods on the controller package
}
