package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	controller "github.com/odnielgonzalez/bedrock-manager-controller/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

func TestReconcileAgent_CreateAgent(t *testing.T) {
	// Setup
	mockClient := NewMockBedrockClient()
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Scheme:        runtime.NewScheme(),
		Recorder:      recorder,
	}

	// Create test resource
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-agent",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: controller.BedrockResourceSpec{
			Type: "agent",
			Agent: &controller.AgentSpec{
				Name:            "TestAgent",
				Description:     stringPtr("Test agent description"),
				FoundationModel: stringPtr("anthropic.claude-3-haiku-20240307-v1:0"),
				Instruction:     stringPtr("You are a test agent"),
				AutoPrepare:     false,
			},
		},
	}

	// Add finalizer to simulate real scenario
	resource.Finalizers = []string{controller.BedrockResourceFinalizer}

	ctx := context.Background()

	// Execute
	result := reconciler.Reconcile(ctx, resource)

	// Verify
	if result.Error != nil {
		t.Errorf("Expected no error, got: %v", result.Error)
	}

	if result.Requeue {
		t.Errorf("Expected no requeue, got: %v", result.Requeue)
	}

	// Check that agent was created
	if resource.Status.AgentId == nil {
		t.Error("Expected AgentId to be set")
	}

	if resource.Status.AgentArn == nil {
		t.Error("Expected AgentArn to be set")
	}

	if resource.Status.Phase != controller.PhaseCreated {
		t.Errorf("Expected phase to be %s, got: %s", controller.PhaseCreated, resource.Status.Phase)
	}

	// Check conditions
	readyCondition := getCondition(resource.Status.Conditions, controller.ConditionReady)
	if readyCondition == nil {
		t.Error("Expected Ready condition to be set")
	} else if readyCondition.Status != metav1.ConditionTrue {
		t.Errorf("Expected Ready condition to be True, got: %s", readyCondition.Status)
	}

	syncedCondition := getCondition(resource.Status.Conditions, controller.ConditionSynced)
	if syncedCondition == nil {
		t.Error("Expected Synced condition to be set")
	} else if syncedCondition.Status != metav1.ConditionTrue {
		t.Errorf("Expected Synced condition to be True, got: %s", syncedCondition.Status)
	}
}

func TestReconcileAgent_CreateAgentWithAutoPrepare(t *testing.T) {
	// Setup
	mockClient := NewMockBedrockClient()
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Scheme:        runtime.NewScheme(),
		Recorder:      recorder,
	}

	// Create test resource with auto-prepare enabled
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-agent-autoprepare",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: controller.BedrockResourceSpec{
			Type: "agent",
			Agent: &controller.AgentSpec{
				Name:            "TestAgentAutoPrepare",
				Description:     stringPtr("Test agent with auto-prepare"),
				FoundationModel: stringPtr("anthropic.claude-3-haiku-20240307-v1:0"),
				Instruction:     stringPtr("You are a test agent with auto-prepare"),
				AutoPrepare:     true,
			},
		},
	}

	// Add finalizer
	resource.Finalizers = []string{controller.BedrockResourceFinalizer}

	ctx := context.Background()

	// Execute
	result := reconciler.Reconcile(ctx, resource)

	// Verify
	if result.Error != nil {
		t.Errorf("Expected no error, got: %v", result.Error)
	}

	// Check that agent was prepared
	if resource.Status.Phase != controller.PhasePrepared {
		t.Errorf("Expected phase to be %s, got: %s", controller.PhasePrepared, resource.Status.Phase)
	}

	// Check prepared condition
	preparedCondition := getCondition(resource.Status.Conditions, controller.ConditionPrepared)
	if preparedCondition == nil {
		t.Error("Expected Prepared condition to be set")
	} else if preparedCondition.Status != metav1.ConditionTrue {
		t.Errorf("Expected Prepared condition to be True, got: %s", preparedCondition.Status)
	}
}

func TestReconcileAgent_UpdateAgent(t *testing.T) {
	// Setup
	mockClient := NewMockBedrockClient()
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Scheme:        runtime.NewScheme(),
		Recorder:      recorder,
	}

	// Pre-create an agent in the mock
	agentId := "existing-agent-1"
	agentArn := "arn:aws:bedrock:us-east-1:123456789012:agent/existing-agent-1"
	mockClient.agents[agentId] = &types.Agent{
		AgentId:      &agentId,
		AgentArn:     &agentArn,
		AgentName:    stringPtr("OldAgentName"),
		AgentStatus:  types.AgentStatusNotPrepared,
		AgentVersion: stringPtr("DRAFT"),
		Description:  stringPtr("Old description"),
	}

	// Create test resource with existing agent ID
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-agent-update",
			Namespace:  "default",
			Generation: 2,
		},
		Spec: controller.BedrockResourceSpec{
			Type: "agent",
			Agent: &controller.AgentSpec{
				Name:            "NewAgentName",               // Changed name
				Description:     stringPtr("New description"), // Changed description
				FoundationModel: stringPtr("anthropic.claude-3-haiku-20240307-v1:0"),
				Instruction:     stringPtr("You are an updated test agent"),
				AutoPrepare:     false,
			},
		},
		Status: controller.BedrockResourceStatus{
			AgentId:  &agentId,
			AgentArn: &agentArn,
		},
	}

	// Add finalizer
	resource.Finalizers = []string{controller.BedrockResourceFinalizer}

	ctx := context.Background()

	// Execute
	result := reconciler.Reconcile(ctx, resource)

	// Verify
	if result.Error != nil {
		t.Errorf("Expected no error, got: %v", result.Error)
	}

	// Check that agent was updated in mock
	updatedAgent := mockClient.agents[agentId]
	if *updatedAgent.AgentName != "NewAgentName" {
		t.Errorf("Expected agent name to be updated to 'NewAgentName', got: %s", *updatedAgent.AgentName)
	}

	if *updatedAgent.Description != "New description" {
		t.Errorf("Expected description to be updated to 'New description', got: %s", *updatedAgent.Description)
	}
}

func TestReconcileAgent_CreateAgentError(t *testing.T) {
	// Setup
	mockClient := NewMockBedrockClient()
	mockClient.SetCreateAgentError(fmt.Errorf("AWS API error"))

	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Scheme:        runtime.NewScheme(),
		Recorder:      recorder,
	}

	// Create test resource
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-agent-error",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: controller.BedrockResourceSpec{
			Type: "agent",
			Agent: &controller.AgentSpec{
				Name:        "TestAgentError",
				AutoPrepare: false,
			},
		},
	}

	// Add finalizer
	resource.Finalizers = []string{controller.BedrockResourceFinalizer}

	ctx := context.Background()

	// Execute
	result := reconciler.Reconcile(ctx, resource)

	// Verify
	if result.Error == nil {
		t.Error("Expected error, got nil")
	}

	if resource.Status.Phase != controller.PhaseFailed {
		t.Errorf("Expected phase to be %s, got: %s", controller.PhaseFailed, resource.Status.Phase)
	}

	// Check conditions
	readyCondition := getCondition(resource.Status.Conditions, controller.ConditionReady)
	if readyCondition == nil {
		t.Error("Expected Ready condition to be set")
	} else if readyCondition.Status != metav1.ConditionFalse {
		t.Errorf("Expected Ready condition to be False, got: %s", readyCondition.Status)
	}

	if result.RequeueAfter != 30*time.Second {
		t.Errorf("Expected RequeueAfter to be 30s, got: %v", result.RequeueAfter)
	}
}

func TestReconcileAgent_DeleteAgent(t *testing.T) {
	// Setup
	mockClient := NewMockBedrockClient()
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Scheme:        runtime.NewScheme(),
		Recorder:      recorder,
	}

	// Pre-create an agent in the mock
	agentId := "delete-agent-1"
	agentArn := "arn:aws:bedrock:us-east-1:123456789012:agent/delete-agent-1"
	mockClient.agents[agentId] = &types.Agent{
		AgentId:      &agentId,
		AgentArn:     &agentArn,
		AgentName:    stringPtr("AgentToDelete"),
		AgentStatus:  types.AgentStatusNotPrepared,
		AgentVersion: stringPtr("DRAFT"),
	}

	// Create test resource with deletion timestamp
	now := metav1.NewTime(time.Now())
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-agent-delete",
			Namespace:         "default",
			Generation:        1,
			DeletionTimestamp: &now,
		},
		Spec: controller.BedrockResourceSpec{
			Type: "agent",
			Agent: &controller.AgentSpec{
				Name:        "AgentToDelete",
				AutoPrepare: false,
			},
		},
		Status: controller.BedrockResourceStatus{
			AgentId:  &agentId,
			AgentArn: &agentArn,
		},
	}

	// Add finalizer
	resource.Finalizers = []string{controller.BedrockResourceFinalizer}

	ctx := context.Background()

	// Execute
	result := reconciler.Reconcile(ctx, resource)

	// Verify
	if result.Error != nil {
		t.Errorf("Expected no error, got: %v", result.Error)
	}

	// Check that agent was deleted from mock
	_, exists := mockClient.agents[agentId]
	if exists {
		t.Error("Expected agent to be deleted from mock")
	}

	// Check that finalizer was removed
	if len(resource.Finalizers) != 0 {
		t.Errorf("Expected finalizers to be empty, got: %v", resource.Finalizers)
	}
}

func TestReconcileInferenceProfile_Implemented(t *testing.T) {
	// Setup
	mockClient := NewMockBedrockClient()
	recorder := record.NewFakeRecorder(10)
	reconciler := &controller.Reconciler{
		BedrockClient: mockClient,
		Scheme:        runtime.NewScheme(),
		Recorder:      recorder,
	}

	// Create test resource for inference profile
	resource := &controller.BedrockResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-inference-profile",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: controller.BedrockResourceSpec{
			Type: "inferenceProfile",
			InferenceProfile: &controller.InferenceProfileSpec{
				Name: "TestInferenceProfile",
				ModelSource: controller.InferenceProfileModelSource{
					CopyFrom: "arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-micro-v1:0",
				},
			},
		},
	}

	// Add finalizer
	resource.Finalizers = []string{controller.BedrockResourceFinalizer}

	ctx := context.Background()

	// Execute
	result := reconciler.Reconcile(ctx, resource)

	// Verify inference profile is now implemented and works
	if result.Error != nil {
		t.Errorf("Expected inference profile to work, got error: %v", result.Error)
	}

	// Should call create inference profile
	if !mockClient.CreateInferenceProfileCalled {
		t.Error("Expected CreateInferenceProfile to be called")
	}

	// Check that status was updated
	if resource.Status.InferenceProfileId == nil {
		t.Error("Expected InferenceProfileId to be set in status")
	}

	if resource.Status.Phase != controller.PhasePrepared {
		t.Errorf("Expected phase to be Prepared, got: %s", resource.Status.Phase)
	}

	// Check conditions
	readyCondition := getCondition(resource.Status.Conditions, controller.ConditionReady)
	if readyCondition == nil {
		t.Error("Expected Ready condition to be set")
	} else if readyCondition.Status != metav1.ConditionTrue {
		t.Errorf("Expected Ready condition to be True, got: %s", readyCondition.Status)
	}
}

func TestAgentNeedsUpdate(t *testing.T) {
	tests := []struct {
		name     string
		spec     *controller.AgentSpec
		agent    *types.Agent
		expected bool
	}{
		{
			name: "no changes needed",
			spec: &controller.AgentSpec{
				Name:        "TestAgent",
				Description: stringPtr("Test description"),
			},
			agent: &types.Agent{
				AgentName:   stringPtr("TestAgent"),
				Description: stringPtr("Test description"),
			},
			expected: false,
		},
		{
			name: "name changed",
			spec: &controller.AgentSpec{
				Name:        "NewTestAgent",
				Description: stringPtr("Test description"),
			},
			agent: &types.Agent{
				AgentName:   stringPtr("TestAgent"),
				Description: stringPtr("Test description"),
			},
			expected: true,
		},
		{
			name: "description changed",
			spec: &controller.AgentSpec{
				Name:        "TestAgent",
				Description: stringPtr("New test description"),
			},
			agent: &types.Agent{
				AgentName:   stringPtr("TestAgent"),
				Description: stringPtr("Test description"),
			},
			expected: true,
		},
		{
			name: "description added",
			spec: &controller.AgentSpec{
				Name:        "TestAgent",
				Description: stringPtr("New test description"),
			},
			agent: &types.Agent{
				AgentName:   stringPtr("TestAgent"),
				Description: nil,
			},
			expected: true,
		},
		{
			name: "description removed",
			spec: &controller.AgentSpec{
				Name:        "TestAgent",
				Description: nil,
			},
			agent: &types.Agent{
				AgentName:   stringPtr("TestAgent"),
				Description: stringPtr("Test description"),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := controller.AgentNeedsUpdate(tt.spec, tt.agent)
			if result != tt.expected {
				t.Errorf("agentNeedsUpdate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Helper functions

func stringPtr(s string) *string {
	return &s
}

func getCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}
