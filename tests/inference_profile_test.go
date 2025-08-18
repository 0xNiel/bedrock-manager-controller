package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	controller "github.com/odnielgonzalez/bedrock-manager-controller/controller"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

func TestInferenceProfileCreation(t *testing.T) {
	tests := []struct {
		name          string
		resource      *controller.BedrockResource
		mockError     error
		expectSuccess bool
		description   string
	}{
		{
			name: "successful_creation",
			resource: &controller.BedrockResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-profile",
					Namespace: "default",
				},
				Spec: controller.BedrockResourceSpec{
					Type: "inferenceProfile",
					InferenceProfile: &controller.InferenceProfileSpec{
						Name:        "TestProfile",
						Description: aws.String("Test inference profile"),
						ModelSource: controller.InferenceProfileModelSource{
							CopyFrom: "arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-micro-v1:0",
						},
						Tags: map[string]string{
							"Environment": "test",
							"Team":        "ai-platform",
						},
					},
				},
			},
			mockError:     nil,
			expectSuccess: true,
			description:   "Should successfully create inference profile",
		},
		{
			name: "creation_with_aws_error",
			resource: &controller.BedrockResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-profile-error",
					Namespace: "default",
				},
				Spec: controller.BedrockResourceSpec{
					Type: "inferenceProfile",
					InferenceProfile: &controller.InferenceProfileSpec{
						Name: "TestProfile",
						ModelSource: controller.InferenceProfileModelSource{
							CopyFrom: "arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-micro-v1:0",
						},
					},
				},
			},
			mockError:     errors.New("AWS API error"),
			expectSuccess: false,
			description:   "Should handle AWS API errors during creation",
		},
		{
			name: "missing_spec",
			resource: &controller.BedrockResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-profile-missing-spec",
					Namespace: "default",
				},
				Spec: controller.BedrockResourceSpec{
					Type: "inferenceProfile",
					// InferenceProfile is nil
				},
			},
			mockError:     nil,
			expectSuccess: false,
			description:   "Should fail when inference profile spec is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := NewMockBedrockClient()
			if tt.mockError != nil {
				mockClient.SetCreateInferenceProfileError(tt.mockError)
			}

			// Create fake recorder
			recorder := record.NewFakeRecorder(10)

			// Create reconciler (without RestClient for unit tests)
			reconciler := &controller.Reconciler{
				BedrockClient: mockClient,
				Recorder:      recorder,
			}

			// Test the reconcile function
			ctx := context.Background()
			result := reconciler.Reconcile(ctx, tt.resource)

			if tt.expectSuccess {
				require.NoError(t, result.Error, "Expected reconciliation to succeed for %s", tt.description)

				if tt.resource.Spec.InferenceProfile != nil {
					// Verify create was called
					require.True(t, mockClient.CreateInferenceProfileCalled, "CreateInferenceProfile should be called")

					// Check input parameters
					input := mockClient.CreateInferenceProfileInput
					require.NotNil(t, input, "Input should be captured")
					require.Equal(t, tt.resource.Spec.InferenceProfile.Name, *input.InferenceProfileName)

					// Check status was updated
					require.NotNil(t, tt.resource.Status.InferenceProfileId, "InferenceProfileId should be set")
					require.NotNil(t, tt.resource.Status.InferenceProfileArn, "InferenceProfileArn should be set")
					require.Equal(t, controller.PhasePrepared, tt.resource.Status.Phase, "Phase should be Prepared")
				}
			} else {
				require.Error(t, result.Error, "Expected reconciliation to fail for %s", tt.description)
			}
		})
	}
}

func TestInferenceProfileDeletion(t *testing.T) {
	tests := []struct {
		name          string
		deleteError   error
		expectSuccess bool
		description   string
	}{
		{
			name:          "successful_deletion",
			deleteError:   nil,
			expectSuccess: true,
			description:   "Should successfully delete inference profile",
		},
		{
			name:          "profile_already_deleted_404",
			deleteError:   errors.New("operation error Bedrock: DeleteInferenceProfile, https response error StatusCode: 404, ResourceNotFoundException: Failed to retrieve resource because it doesn't exist"),
			expectSuccess: true,
			description:   "Should treat 404 as successful deletion",
		},
		{
			name:          "profile_not_found_exception",
			deleteError:   errors.New("ResourceNotFoundException: Inference profile not found"),
			expectSuccess: true,
			description:   "Should treat ResourceNotFoundException as successful deletion",
		},
		{
			name:          "other_aws_error",
			deleteError:   errors.New("AccessDeniedException: Insufficient permissions"),
			expectSuccess: false,
			description:   "Should fail on other AWS errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := NewMockBedrockClient()

			// Create resource being deleted
			profileId := "test-profile-123"

			// For successful deletion test, pre-create the profile in the mock
			if tt.deleteError == nil {
				mockClient.AddInferenceProfile(&bedrocktypes.InferenceProfileSummary{
					InferenceProfileId:   &profileId,
					InferenceProfileArn:  aws.String("arn:aws:bedrock:us-east-1:123456789012:inference-profile/test-profile-123"),
					InferenceProfileName: aws.String("TestProfile"),
					Status:               bedrocktypes.InferenceProfileStatusActive,
				})
			}

			mockClient.SetDeleteInferenceProfileError(tt.deleteError)

			// Create fake recorder
			recorder := record.NewFakeRecorder(10)

			// Create reconciler
			reconciler := &controller.Reconciler{
				BedrockClient: mockClient,
				Recorder:      recorder,
			}
			resource := &controller.BedrockResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-profile",
					Namespace:         "default",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{controller.BedrockResourceFinalizer},
				},
				Spec: controller.BedrockResourceSpec{
					Type: "inferenceProfile",
					InferenceProfile: &controller.InferenceProfileSpec{
						Name: "TestProfile",
						ModelSource: controller.InferenceProfileModelSource{
							CopyFrom: "arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-micro-v1:0",
						},
					},
				},
				Status: controller.BedrockResourceStatus{
					Phase:              controller.PhaseDeleting,
					InferenceProfileId: &profileId,
				},
			}

			// Test the reconcile deletion flow
			ctx := context.Background()
			result := reconciler.Reconcile(ctx, resource)

			if tt.expectSuccess {
				require.NoError(t, result.Error, "Reconcile deletion should succeed for %s", tt.description)

				// Should not requeue after successful deletion
				require.False(t, result.Requeue, "Should not requeue after successful deletion")
				require.Zero(t, result.RequeueAfter, "Should not requeue after successful deletion")
			} else {
				require.Error(t, result.Error, "Reconcile deletion should fail for %s", tt.description)
			}

			// Verify delete was called
			require.True(t, mockClient.DeleteInferenceProfileCalled, "DeleteInferenceProfile should be called")
			require.Equal(t, profileId, *mockClient.DeleteInferenceProfileInput.InferenceProfileIdentifier)
		})
	}
}

func TestInferenceProfileConversionHelpers(t *testing.T) {
	t.Run("convertInferenceProfileSpecToCreateInput", func(t *testing.T) {
		spec := &controller.InferenceProfileSpec{
			Name:        "TestProfile",
			Description: aws.String("Test description"),
			ModelSource: controller.InferenceProfileModelSource{
				CopyFrom: "arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-micro-v1:0",
			},
			Tags: map[string]string{
				"Environment": "test",
				"Team":        "ai-platform",
			},
		}

		// We need to import the helper function or make it public
		// For now, let's test indirectly through the mock
		mockClient := NewMockBedrockClient()
		recorder := record.NewFakeRecorder(10)
		reconciler := &controller.Reconciler{
			BedrockClient: mockClient,
			Recorder:      recorder,
		}

		resource := &controller.BedrockResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-profile",
				Namespace: "default",
			},
			Spec: controller.BedrockResourceSpec{
				Type:             "inferenceProfile",
				InferenceProfile: spec,
			},
		}

		ctx := context.Background()
		result := reconciler.Reconcile(ctx, resource)

		require.NoError(t, result.Error, "Should succeed")
		require.True(t, mockClient.CreateInferenceProfileCalled, "Should call create")

		input := mockClient.CreateInferenceProfileInput
		require.Equal(t, spec.Name, *input.InferenceProfileName)
		require.Equal(t, *spec.Description, *input.Description)
		require.Len(t, input.Tags, 2, "Should have 2 tags")
	})
}

func TestInferenceProfileReconciliation(t *testing.T) {
	t.Run("existing_profile_status_update", func(t *testing.T) {
		// Create mock client with existing profile
		mockClient := NewMockBedrockClient()
		profileId := "existing-profile-123"
		profileArn := "arn:aws:bedrock:us-east-1:123456789012:inference-profile/existing-profile-123"

		// Add existing profile to mock
		mockClient.AddInferenceProfile(&bedrocktypes.InferenceProfileSummary{
			InferenceProfileId:   &profileId,
			InferenceProfileArn:  &profileArn,
			InferenceProfileName: aws.String("ExistingProfile"),
			Status:               bedrocktypes.InferenceProfileStatusActive,
		})

		recorder := record.NewFakeRecorder(10)
		reconciler := &controller.Reconciler{
			BedrockClient: mockClient,
			Recorder:      recorder,
		}

		resource := &controller.BedrockResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-profile",
				Namespace: "default",
			},
			Spec: controller.BedrockResourceSpec{
				Type: "inferenceProfile",
				InferenceProfile: &controller.InferenceProfileSpec{
					Name: "ExistingProfile",
					ModelSource: controller.InferenceProfileModelSource{
						CopyFrom: "arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-micro-v1:0",
					},
				},
			},
			Status: controller.BedrockResourceStatus{
				InferenceProfileId: &profileId,
			},
		}

		ctx := context.Background()
		result := reconciler.Reconcile(ctx, resource)

		require.NoError(t, result.Error, "Should succeed for existing profile")
		require.True(t, mockClient.GetInferenceProfileCalled, "Should call get")
		require.False(t, mockClient.CreateInferenceProfileCalled, "Should not call create")

		// Check status was updated
		require.Equal(t, controller.PhasePrepared, resource.Status.Phase, "Phase should be Prepared")
		require.Equal(t, profileId, *resource.Status.InferenceProfileId)
		require.Equal(t, profileArn, *resource.Status.InferenceProfileArn)
	})
}

func TestInferenceProfileNoInfiniteLoop(t *testing.T) {
	// Test that ensures we don't create multiple inference profiles
	// when the status is properly persisted (regression test for the infinite loop bug)

	mockClient := &MockBedrockClient{}
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Recorder:      recorder,
		// Note: RestClient is nil in unit tests, which is expected
	}

	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-inference-profile",
			Namespace: "default",
		},
		Spec: controller.BedrockResourceSpec{
			Type: "inferenceProfile",
			InferenceProfile: &controller.InferenceProfileSpec{
				Name:        "TestProfile",
				Description: aws.String("Test profile for infinite loop regression test"),
				ModelSource: controller.InferenceProfileModelSource{
					CopyFrom: "amazon.nova-micro-v1:0",
				},
			},
		},
	}

	ctx := context.Background()

	// First reconciliation - should create the inference profile
	result1 := reconciler.Reconcile(ctx, resource)
	require.NoError(t, result1.Error)
	require.False(t, result1.Requeue)
	require.Equal(t, time.Duration(0), result1.RequeueAfter)

	// Verify inference profile was created
	require.True(t, mockClient.CreateInferenceProfileCalled)
	require.Equal(t, 1, len(mockClient.GetInferenceProfiles()))
	require.NotNil(t, resource.Status.InferenceProfileId)

	// Store the ID from first creation
	firstProfileId := *resource.Status.InferenceProfileId

	// Reset the mock client call tracking (but keep the created profile in state)
	mockClient.CreateInferenceProfileCalled = false

	// Second reconciliation - should NOT create another inference profile
	// This simulates what happens when Kubernetes re-reconciles the same resource
	result2 := reconciler.Reconcile(ctx, resource)
	require.NoError(t, result2.Error)
	require.False(t, result2.Requeue)
	require.Equal(t, time.Duration(0), result2.RequeueAfter)

	// Verify NO additional inference profile was created
	require.False(t, mockClient.CreateInferenceProfileCalled, "CreateInferenceProfile should not be called on second reconciliation")
	require.Equal(t, 1, len(mockClient.GetInferenceProfiles()), "Should still have exactly 1 inference profile")
	require.NotNil(t, resource.Status.InferenceProfileId)
	require.Equal(t, firstProfileId, *resource.Status.InferenceProfileId, "Inference profile ID should remain the same")

	// Third reconciliation - verify it's still stable
	mockClient.CreateInferenceProfileCalled = false
	result3 := reconciler.Reconcile(ctx, resource)
	require.NoError(t, result3.Error)
	require.False(t, result3.Requeue)

	// Still no additional creation
	require.False(t, mockClient.CreateInferenceProfileCalled, "CreateInferenceProfile should not be called on third reconciliation")
	require.Equal(t, 1, len(mockClient.GetInferenceProfiles()), "Should still have exactly 1 inference profile")
	require.Equal(t, firstProfileId, *resource.Status.InferenceProfileId, "Inference profile ID should remain stable")

	// Verify the status fields are properly set
	require.NotNil(t, resource.Status.InferenceProfileArn)
	require.Equal(t, controller.PhasePrepared, resource.Status.Phase)
	require.Equal(t, firstProfileId, *resource.Status.ResourceId)
	require.NotNil(t, resource.Status.ResourceArn)
}

func TestInferenceProfileStatusPersistence(t *testing.T) {
	// Test that simulates multiple reconciliation cycles with fresh resource objects
	// (like what happens in the real controller when resources are fetched from cache)

	mockClient := &MockBedrockClient{}
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Recorder:      recorder,
	}

	// Create initial resource (simulates fresh fetch from Kubernetes cache)
	createResource := func() *controller.BedrockResource {
		return &controller.BedrockResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-persistence",
				Namespace: "default",
			},
			Spec: controller.BedrockResourceSpec{
				Type: "inferenceProfile",
				InferenceProfile: &controller.InferenceProfileSpec{
					Name:        "PersistenceTest",
					Description: aws.String("Test profile for status persistence"),
					ModelSource: controller.InferenceProfileModelSource{
						CopyFrom: "amazon.nova-micro-v1:0",
					},
				},
			},
		}
	}

	ctx := context.Background()

	// First reconciliation with fresh resource
	resource1 := createResource()
	result1 := reconciler.Reconcile(ctx, resource1)
	require.NoError(t, result1.Error)
	require.True(t, mockClient.CreateInferenceProfileCalled)
	require.NotNil(t, resource1.Status.InferenceProfileId)

	createdProfileId := *resource1.Status.InferenceProfileId
	createdProfileArn := *resource1.Status.InferenceProfileArn

	// Second reconciliation with fresh resource but simulating that status was persisted
	resource2 := createResource()
	// Simulate that Kubernetes has persisted the status from the previous reconciliation
	resource2.Status.InferenceProfileId = aws.String(createdProfileId)
	resource2.Status.InferenceProfileArn = aws.String(createdProfileArn)
	resource2.Status.ResourceId = aws.String(createdProfileId)
	resource2.Status.ResourceArn = aws.String(createdProfileArn)
	resource2.Status.Phase = controller.PhasePrepared

	// Add the profile to mock client's state (simulates it exists in AWS)
	mockClient.AddInferenceProfile(&bedrocktypes.InferenceProfileSummary{
		InferenceProfileId:  aws.String(createdProfileId),
		InferenceProfileArn: aws.String(createdProfileArn),
		Status:              bedrocktypes.InferenceProfileStatusActive,
	})

	// Reset call tracking
	mockClient.CreateInferenceProfileCalled = false

	result2 := reconciler.Reconcile(ctx, resource2)
	require.NoError(t, result2.Error)

	// Should NOT create another profile since it already exists
	require.False(t, mockClient.CreateInferenceProfileCalled, "Should not create new profile when ID exists in status")
	require.Equal(t, createdProfileId, *resource2.Status.InferenceProfileId)
	require.Equal(t, controller.PhasePrepared, resource2.Status.Phase)
}
