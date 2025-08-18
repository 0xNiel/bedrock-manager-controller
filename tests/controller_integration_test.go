package tests

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	controller "github.com/odnielgonzalez/bedrock-manager-controller/controller"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

// TestInfiniteLoopReproduction tests the infinite loop issue by simulating
// multiple reconciliation cycles like the real controller
func TestInfiniteLoopReproduction(t *testing.T) {
	// This test simulates the real controller pattern where:
	// 1. Each reconciliation gets a fresh copy from cache
	// 2. Status updates happen after reconciliation
	// 3. The next reconciliation should see the updated status

	mockClient := &MockBedrockClient{}
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Recorder:      recorder,
		// RestClient is nil to simulate unit test environment
	}

	// Simulate the original resource from cache (fresh each time)
	createFreshResource := func() *controller.BedrockResource {
		return &controller.BedrockResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "test-infinite-loop",
				Namespace:       "default",
				ResourceVersion: "1000", // Simulate K8s resource version
				Generation:      1,
				Finalizers:      []string{controller.BedrockResourceFinalizer},
			},
			Spec: controller.BedrockResourceSpec{
				Type: "inferenceProfile",
				InferenceProfile: &controller.InferenceProfileSpec{
					Name:        "TestProfile",
					Description: aws.String("Test for infinite loop"),
					ModelSource: controller.InferenceProfileModelSource{
						CopyFrom: "amazon.nova-micro-v1:0",
					},
				},
			},
		}
	}

	ctx := context.Background()
	var savedStatus controller.BedrockResourceStatus

	// CYCLE 1: First reconciliation (should create profile)
	t.Run("Cycle1_CreateProfile", func(t *testing.T) {
		resource1 := createFreshResource()

		result1 := reconciler.Reconcile(ctx, resource1)
		require.NoError(t, result1.Error)

		// Verify profile was created
		require.True(t, mockClient.CreateInferenceProfileCalled)
		require.Equal(t, 1, len(mockClient.GetInferenceProfiles()))
		require.NotNil(t, resource1.Status.InferenceProfileId)

		// Save the status (simulates what UpdateResourceStatus would persist)
		savedStatus = resource1.Status

		t.Logf("Created profile with ID: %s", *resource1.Status.InferenceProfileId)
	})

	// Reset call tracking but keep the created profile in mock state
	mockClient.CreateInferenceProfileCalled = false

	// CYCLE 2: Second reconciliation (should NOT create another profile)
	// This simulates getting a fresh resource from cache with persisted status
	t.Run("Cycle2_ShouldNotCreateAnother", func(t *testing.T) {
		resource2 := createFreshResource()
		// Simulate that Kubernetes persisted the status from cycle 1
		resource2.Status = savedStatus

		result2 := reconciler.Reconcile(ctx, resource2)
		require.NoError(t, result2.Error)

		// Should NOT create another profile
		require.False(t, mockClient.CreateInferenceProfileCalled,
			"Second reconciliation should not create another profile when status has InferenceProfileId")
		require.Equal(t, 1, len(mockClient.GetInferenceProfiles()),
			"Should still have exactly 1 profile")

		t.Logf("Second cycle correctly reused existing profile: %s", *resource2.Status.InferenceProfileId)
	})

	// CYCLE 3: Third reconciliation (final verification)
	t.Run("Cycle3_StillStable", func(t *testing.T) {
		resource3 := createFreshResource()
		resource3.Status = savedStatus

		mockClient.CreateInferenceProfileCalled = false // Reset again

		result3 := reconciler.Reconcile(ctx, resource3)
		require.NoError(t, result3.Error)

		// Still should not create another profile
		require.False(t, mockClient.CreateInferenceProfileCalled)
		require.Equal(t, 1, len(mockClient.GetInferenceProfiles()))

		t.Logf("Third cycle still stable with profile: %s", *resource3.Status.InferenceProfileId)
	})
}

// TestFinalizerStuckReproduction tests the finalizer stuck issue
func TestFinalizerStuckReproduction(t *testing.T) {
	mockClient := &MockBedrockClient{}
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Recorder:      recorder,
		// RestClient is nil - this might be part of the problem!
	}

	// Create and set up a resource with an existing profile
	profileId := "test-finalizer-profile"
	profileArn := "arn:aws:bedrock:us-east-1:123456789012:inference-profile/" + profileId

	// Add profile to mock client state
	mockClient.AddInferenceProfile(&bedrocktypes.InferenceProfileSummary{
		InferenceProfileId:  aws.String(profileId),
		InferenceProfileArn: aws.String(profileArn),
		Status:              bedrocktypes.InferenceProfileStatusActive,
	})

	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-finalizer",
			Namespace:         "default",
			ResourceVersion:   "2000",
			Generation:        1,
			DeletionTimestamp: &metav1.Time{Time: time.Now()}, // Marked for deletion
			Finalizers:        []string{controller.BedrockResourceFinalizer},
		},
		Spec: controller.BedrockResourceSpec{
			Type: "inferenceProfile",
			InferenceProfile: &controller.InferenceProfileSpec{
				Name:        "TestProfile",
				Description: aws.String("Test for finalizer"),
				ModelSource: controller.InferenceProfileModelSource{
					CopyFrom: "amazon.nova-micro-v1:0",
				},
			},
		},
		Status: controller.BedrockResourceStatus{
			InferenceProfileId:  aws.String(profileId),
			InferenceProfileArn: aws.String(profileArn),
			ResourceId:          aws.String(profileId),
			ResourceArn:         aws.String(profileArn),
			Phase:               controller.PhasePrepared,
		},
	}

	ctx := context.Background()

	t.Run("DeletionShouldSucceed", func(t *testing.T) {
		// Verify profile exists before deletion
		require.Equal(t, 1, len(mockClient.GetInferenceProfiles()))

		result := reconciler.Reconcile(ctx, resource)

		// Should not error
		require.NoError(t, result.Error)

		// Should delete the AWS profile
		require.True(t, mockClient.DeleteInferenceProfileCalled)

		// Profile should be removed from mock state
		require.Equal(t, 0, len(mockClient.GetInferenceProfiles()))

		// Finalizer should be removed from the resource
		require.NotContains(t, resource.Finalizers, controller.BedrockResourceFinalizer,
			"Finalizer should be removed after successful deletion")

		t.Logf("Deletion completed successfully, finalizer removed")
	})
}

// TestCacheStaleStatusIssue tests the case where the controller cache
// gives stale status information leading to infinite loops
func TestCacheStaleStatusIssue(t *testing.T) {
	mockClient := &MockBedrockClient{}
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Recorder:      recorder,
	}

	// Simulate the problematic scenario where:
	// 1. Controller creates a profile and updates local status
	// 2. Status update to K8s "succeeds" but status isn't actually persisted
	// 3. Next reconciliation gets stale data from cache (no InferenceProfileId)
	// 4. Controller should now discover the existing profile by name instead of creating another

	profileName := "TestStaleCache"
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-stale-cache",
			Namespace:       "default",
			ResourceVersion: "3000",
			Generation:      1,
			Finalizers:      []string{controller.BedrockResourceFinalizer},
		},
		Spec: controller.BedrockResourceSpec{
			Type: "inferenceProfile",
			InferenceProfile: &controller.InferenceProfileSpec{
				Name:        profileName,
				Description: aws.String("Test for stale cache"),
				ModelSource: controller.InferenceProfileModelSource{
					CopyFrom: "amazon.nova-micro-v1:0",
				},
			},
		},
		// Status is empty (simulating fresh resource from cache)
	}

	ctx := context.Background()

	// First reconciliation - creates profile
	result1 := reconciler.Reconcile(ctx, resource)
	require.NoError(t, result1.Error)
	require.True(t, mockClient.CreateInferenceProfileCalled)
	require.NotNil(t, resource.Status.InferenceProfileId)

	firstProfileId := *resource.Status.InferenceProfileId
	t.Logf("First reconciliation created profile: %s", firstProfileId)

	// Simulate that the profile exists in AWS but the K8s status wasn't persisted
	// (due to optimistic concurrency conflicts or other issues)
	// This means the next reconciliation will see empty status
	resourceWithStaleStatus := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-stale-cache",
			Namespace:       "default",
			ResourceVersion: "3001", // Bumped resource version
			Generation:      1,
			Finalizers:      []string{controller.BedrockResourceFinalizer},
		},
		Spec: controller.BedrockResourceSpec{
			Type: "inferenceProfile",
			InferenceProfile: &controller.InferenceProfileSpec{
				Name:        "TestProfile",
				Description: aws.String("Test for stale cache"),
				ModelSource: controller.InferenceProfileModelSource{
					CopyFrom: "amazon.nova-micro-v1:0",
				},
			},
		},
		// Status is EMPTY again! This simulates the cache serving stale data
		Status: controller.BedrockResourceStatus{},
	}

	// Reset call tracking but keep the created profile in mock state
	mockClient.CreateInferenceProfileCalled = false

	// Update the second resource to use the same profile name
	resourceWithStaleStatus.Spec.InferenceProfile.Name = profileName

	// Second reconciliation with stale cache - this is where the bug occurs
	result2 := reconciler.Reconcile(ctx, resourceWithStaleStatus)
	require.NoError(t, result2.Error)

	// THE BUG: This should NOT create another profile, but it might because
	// the status is empty, making the controller think no profile exists
	if mockClient.CreateInferenceProfileCalled {
		t.Errorf("BUG REPRODUCED: Second reconciliation created another profile when status was stale!")
		t.Errorf("Profile count: %d (should be 1)", len(mockClient.GetInferenceProfiles()))

		// This reproduces the infinite loop issue
		profiles := mockClient.GetInferenceProfiles()
		if len(profiles) > 1 {
			for id, profile := range profiles {
				t.Logf("Profile %s: %+v", id, profile)
			}
		}
	} else {
		t.Logf("Good: Second reconciliation did not create another profile")
		require.Equal(t, 1, len(mockClient.GetInferenceProfiles()))
	}
}
