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

func TestDeletionFlow(t *testing.T) {
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
			description:   "Agent exists and is successfully deleted",
		},
		{
			name:          "agent_already_deleted_404",
			deleteError:   errors.New("operation error Bedrock Agent: DeleteAgent, https response error StatusCode: 404, RequestID: test-id, ResourceNotFoundException: Failed to retrieve resource because it doesn't exist"),
			expectSuccess: true,
			description:   "Agent already deleted (404) should be treated as success",
		},
		{
			name:          "agent_not_found_exception",
			deleteError:   errors.New("ResourceNotFoundException: Agent not found"),
			expectSuccess: true,
			description:   "ResourceNotFoundException should be treated as success",
		},
		{
			name:          "other_aws_error",
			deleteError:   errors.New("AccessDeniedException: Insufficient permissions"),
			expectSuccess: false,
			description:   "Other AWS errors should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := NewMockBedrockClient()

			// Create resource being deleted
			agentId := "test-agent-123"

			// For successful deletion test, pre-create the agent in the mock
			if tt.deleteError == nil {
				mockClient.AddAgent(&types.Agent{
					AgentId:     &agentId,
					AgentName:   aws.String("TestAgent"),
					AgentStatus: types.AgentStatusPrepared,
				})
			}

			mockClient.SetDeleteAgentError(tt.deleteError)

			// Create fake recorder
			recorder := record.NewFakeRecorder(10)

			// Create reconciler (without RestClient for unit tests)
			reconciler := &controller.Reconciler{
				BedrockClient: mockClient,
				Recorder:      recorder,
			}

			resource := &controller.BedrockResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-agent",
					Namespace:         "default",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{controller.BedrockResourceFinalizer},
				},
				Spec: controller.BedrockResourceSpec{
					Type: "agent",
					Agent: &controller.AgentSpec{
						Name: "TestAgent",
					},
				},
				Status: controller.BedrockResourceStatus{
					AgentId: &agentId,
				},
			}

			// Test via the main Reconcile function since deleteAgent is private
			ctx := context.Background()
			result := reconciler.Reconcile(ctx, resource)
			err := result.Error

			if tt.expectSuccess {
				require.NoError(t, err, "Expected deletion to succeed for %s", tt.description)
			} else {
				require.Error(t, err, "Expected deletion to fail for %s", tt.description)
			}

			// Verify delete was called
			require.True(t, mockClient.DeleteAgentCalled, "DeleteAgent should be called")
			require.Equal(t, agentId, *mockClient.DeleteAgentInput.AgentId, "AgentId should match")
		})
	}
}

func TestFinalizerHandling(t *testing.T) {
	tests := []struct {
		name               string
		initialFinalizers  []string
		expectedFinalizers []string
		description        string
	}{
		{
			name:               "remove_bedrock_finalizer",
			initialFinalizers:  []string{controller.BedrockResourceFinalizer},
			expectedFinalizers: []string{},
			description:        "Should remove only the bedrock finalizer",
		},
		{
			name:               "remove_bedrock_among_others",
			initialFinalizers:  []string{"other.finalizer.com", controller.BedrockResourceFinalizer, "another.finalizer.com"},
			expectedFinalizers: []string{"other.finalizer.com", "another.finalizer.com"},
			description:        "Should remove only the bedrock finalizer, leaving others",
		},
		{
			name:               "no_bedrock_finalizer",
			initialFinalizers:  []string{"other.finalizer.com"},
			expectedFinalizers: []string{"other.finalizer.com"},
			description:        "Should not change finalizers if bedrock finalizer not present",
		},
		{
			name:               "empty_finalizers",
			initialFinalizers:  []string{},
			expectedFinalizers: []string{},
			description:        "Should handle empty finalizer list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create resource with initial finalizers
			resource := &controller.BedrockResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-resource",
					Namespace:  "default",
					Finalizers: tt.initialFinalizers,
				},
			}

			// Test finalizer removal logic
			originalFinalizers := make([]string, len(resource.Finalizers))
			copy(originalFinalizers, resource.Finalizers)

			// Apply the removal logic (simulate what happens in handleDeletion)
			resource.Finalizers = removeFinalizer(resource.Finalizers, controller.BedrockResourceFinalizer)

			require.Equal(t, tt.expectedFinalizers, resource.Finalizers, "Finalizers should match expected for %s", tt.description)
		})
	}
}

func TestReconcileDeletion(t *testing.T) {
	// Create mock client that simulates agent already deleted
	mockClient := NewMockBedrockClient()
	mockClient.SetDeleteAgentError(errors.New("ResourceNotFoundException: Agent not found"))

	// Create fake recorder
	recorder := record.NewFakeRecorder(10)

	// Create reconciler
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Recorder:      recorder,
	}

	// Create resource being deleted
	agentId := "test-agent-123"
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-agent",
			Namespace:         "default",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{controller.BedrockResourceFinalizer},
		},
		Spec: controller.BedrockResourceSpec{
			Type: "agent",
			Agent: &controller.AgentSpec{
				Name: "TestAgent",
			},
		},
		Status: controller.BedrockResourceStatus{
			Phase:   controller.PhaseDeleting,
			AgentId: &agentId,
		},
	}

	// Test the reconcile deletion flow
	ctx := context.Background()
	result := reconciler.Reconcile(ctx, resource)

	// Should succeed (no error)
	require.NoError(t, result.Error, "Reconcile deletion should succeed when agent already deleted")

	// Should not requeue
	require.False(t, result.Requeue, "Should not requeue after successful deletion")
	require.Zero(t, result.RequeueAfter, "Should not requeue after successful deletion")

	// Verify DeleteAgent was called
	require.True(t, mockClient.DeleteAgentCalled, "DeleteAgent should be called")

	// Verify events were recorded
	close(recorder.Events)
	var events []string
	for event := range recorder.Events {
		events = append(events, event)
	}

	// Should have a "Deleted" event
	require.Len(t, events, 1, "Should record exactly one event")
	require.Contains(t, events[0], "Normal Deleted Resource deleted successfully", "Should record deletion success event")
}

// Helper function to test finalizer removal (this mirrors the logic in reconcile.go)
func removeFinalizer(finalizers []string, finalizerToRemove string) []string {
	var result []string
	for _, f := range finalizers {
		if f != finalizerToRemove {
			result = append(result, f)
		}
	}
	// Ensure we return an empty slice instead of nil for consistent comparisons
	if result == nil {
		result = []string{}
	}
	return result
}
