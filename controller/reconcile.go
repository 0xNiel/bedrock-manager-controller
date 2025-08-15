package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

// Reconciler handles the reconciliation logic for BedrockResource
type Reconciler struct {
	BedrockClient BedrockClient
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	RestClient    rest.Interface
}

// ReconcileResult represents the result of a reconciliation
type ReconcileResult struct {
	// Requeue indicates whether the request should be requeued
	Requeue bool
	// RequeueAfter indicates how long to wait before requeuing
	RequeueAfter time.Duration
	// Error indicates an error occurred during reconciliation
	Error error
}

// Reconcile handles the main reconciliation logic for a BedrockResource
func (r *Reconciler) Reconcile(ctx context.Context, resource *BedrockResource) ReconcileResult {
	log := klog.FromContext(ctx).WithValues("resource", resource.Name, "namespace", resource.Namespace)
	log.Info("Starting reconciliation")

	// Update status to indicate reconciliation is in progress
	now := metav1.NewTime(time.Now())
	resource.Status.LastSyncTime = &now
	resource.Status.ObservedGeneration = resource.Generation

	// Handle deletion
	if !resource.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, resource)
	}

	// Add finalizer if not present
	if !containsFinalizer(resource, BedrockResourceFinalizer) {
		resource.Finalizers = append(resource.Finalizers, BedrockResourceFinalizer)
		// Update the resource to persist the finalizer
		if err := r.updateResource(ctx, resource); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ReconcileResult{Error: err, RequeueAfter: time.Second * 10}
		}
		return ReconcileResult{Requeue: true}
	}

	// Route to appropriate handler based on resource type
	switch resource.Spec.Type {
	case "agent":
		return r.reconcileAgent(ctx, resource)
	case "inferenceProfile":
		return r.reconcileInferenceProfile(ctx, resource)
	case "knowledgeBase":
		return r.reconcileKnowledgeBase(ctx, resource)
	default:
		err := fmt.Errorf("unknown resource type: %s", resource.Spec.Type)
		r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonValidationError, err.Error())
		r.Recorder.Event(resource, "Warning", "ValidationError", err.Error())
		return ReconcileResult{Error: err}
	}
}

// reconcileAgent handles Agent-specific reconciliation
func (r *Reconciler) reconcileAgent(ctx context.Context, resource *BedrockResource) ReconcileResult {
	log := klog.FromContext(ctx).WithValues("type", "agent")

	if resource.Spec.Agent == nil {
		err := fmt.Errorf("agent spec is required when type is agent")
		r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonValidationError, err.Error())
		return ReconcileResult{Error: err}
	}

	// Check if agent already exists
	var existingAgent *types.Agent
	if resource.Status.AgentId != nil {
		getOutput, err := r.BedrockClient.GetAgent(ctx, &bedrockagent.GetAgentInput{
			AgentId: resource.Status.AgentId,
		})
		if err != nil {
			// Agent might have been deleted outside of Kubernetes
			log.Info("Agent not found in AWS, will recreate", "agentId", *resource.Status.AgentId)
			resource.Status.AgentId = nil
			resource.Status.AgentArn = nil
		} else {
			existingAgent = getOutput.Agent
		}
	}

	if existingAgent == nil {
		// Create new agent
		return r.createAgent(ctx, resource)
	} else {
		// Update existing agent
		return r.updateAgent(ctx, resource, existingAgent)
	}
}

// createAgent creates a new Bedrock Agent
func (r *Reconciler) createAgent(ctx context.Context, resource *BedrockResource) ReconcileResult {
	log := klog.FromContext(ctx).WithValues("action", "create")
	log.Info("Creating new agent")

	// Update status to Creating
	resource.Status.Phase = PhaseCreating
	r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonCreating, "Creating agent in AWS")
	r.updateCondition(resource, ConditionSynced, metav1.ConditionFalse, ReasonCreating, "Creating agent")

	// Convert spec to AWS input
	input := convertAgentSpecToCreateInput(resource.Spec.Agent)

	// Create agent
	output, err := r.BedrockClient.CreateAgent(ctx, input)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create agent: %v", err)
		log.Error(err, "Failed to create agent")
		resource.Status.Phase = PhaseFailed
		r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonAWSError, errMsg)
		r.updateCondition(resource, ConditionSynced, metav1.ConditionFalse, ReasonAWSError, errMsg)
		r.Recorder.Event(resource, "Warning", "CreateFailed", errMsg)
		return ReconcileResult{Error: err, RequeueAfter: 30 * time.Second}
	}

	// Update status with created agent info
	agentStatus := convertAWSAgentToStatus(output.Agent)
	resource.Status.AgentId = agentStatus.AgentId
	resource.Status.AgentArn = agentStatus.AgentArn
	resource.Status.AgentStatus = agentStatus.AgentStatus
	resource.Status.AgentVersion = agentStatus.AgentVersion
	resource.Status.ResourceId = agentStatus.ResourceId
	resource.Status.ResourceArn = agentStatus.ResourceArn
	resource.Status.Phase = PhaseCreated

	r.updateCondition(resource, ConditionSynced, metav1.ConditionTrue, ReasonCreated, "Agent created successfully")
	r.Recorder.Event(resource, "Normal", "Created", fmt.Sprintf("Agent created with ID %s", *agentStatus.AgentId))

	log.Info("Agent created successfully", "agentId", *agentStatus.AgentId)

	// Prepare agent if autoPrepare is enabled
	if resource.Spec.Agent.AutoPrepare {
		return r.prepareAgent(ctx, resource)
	}

	r.updateCondition(resource, ConditionReady, metav1.ConditionTrue, ReasonCreated, "Agent created and ready")
	return ReconcileResult{}
}

// updateAgent updates an existing Bedrock Agent
func (r *Reconciler) updateAgent(ctx context.Context, resource *BedrockResource, existingAgent *types.Agent) ReconcileResult {
	log := klog.FromContext(ctx).WithValues("action", "update", "agentId", *existingAgent.AgentId)

	// Check if update is needed by comparing specs
	if !AgentNeedsUpdate(resource.Spec.Agent, existingAgent) {
		log.V(1).Info("Agent is up to date, no changes needed", "awsAgentStatus", existingAgent.AgentStatus)

		// Update status from AWS agent
		agentStatus := convertAWSAgentToStatus(existingAgent)
		resource.Status.AgentStatus = agentStatus.AgentStatus
		resource.Status.AgentVersion = agentStatus.AgentVersion
		resource.Status.Phase = agentStatus.Phase

		log.V(1).Info("Updated phase from AWS status", "awsStatus", existingAgent.AgentStatus, "newPhase", agentStatus.Phase)

		// Check if agent should be prepared
		if resource.Spec.Agent.AutoPrepare && existingAgent.AgentStatus == types.AgentStatusNotPrepared {
			log.Info("Agent needs preparation", "currentStatus", existingAgent.AgentStatus)
			return r.prepareAgent(ctx, resource)
		}

		r.updateCondition(resource, ConditionSynced, metav1.ConditionTrue, ReasonReconciling, "Agent is up to date")
		r.updateCondition(resource, ConditionReady, metav1.ConditionTrue, ReasonReconciling, "Agent is ready")
		return ReconcileResult{}
	}

	log.Info("Updating agent")
	resource.Status.Phase = PhaseUpdating
	r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonUpdating, "Updating agent in AWS")

	// Convert spec to AWS update input
	input := convertAgentSpecToUpdateInput(resource.Spec.Agent, *existingAgent.AgentId)

	// Update agent
	output, err := r.BedrockClient.UpdateAgent(ctx, input)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to update agent: %v", err)
		log.Error(err, "Failed to update agent")
		resource.Status.Phase = PhaseFailed
		r.updateCondition(resource, ConditionSynced, metav1.ConditionFalse, ReasonAWSError, errMsg)
		r.Recorder.Event(resource, "Warning", "UpdateFailed", errMsg)
		return ReconcileResult{Error: err, RequeueAfter: 30 * time.Second}
	}

	// Update status
	agentStatus := convertAWSAgentToStatus(output.Agent)
	resource.Status.AgentStatus = agentStatus.AgentStatus
	resource.Status.AgentVersion = agentStatus.AgentVersion
	resource.Status.Phase = agentStatus.Phase

	r.updateCondition(resource, ConditionSynced, metav1.ConditionTrue, ReasonReconciling, "Agent updated successfully")
	r.Recorder.Event(resource, "Normal", "Updated", "Agent updated successfully")

	log.Info("Agent updated successfully")

	// Prepare agent if autoPrepare is enabled
	if resource.Spec.Agent.AutoPrepare {
		return r.prepareAgent(ctx, resource)
	}

	r.updateCondition(resource, ConditionReady, metav1.ConditionTrue, ReasonReconciling, "Agent updated and ready")
	return ReconcileResult{}
}

// prepareAgent prepares a Bedrock Agent
func (r *Reconciler) prepareAgent(ctx context.Context, resource *BedrockResource) ReconcileResult {
	log := klog.FromContext(ctx).WithValues("action", "prepare", "agentId", *resource.Status.AgentId)
	log.Info("Preparing agent")

	resource.Status.Phase = PhasePreparing
	r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonPreparing, "Preparing agent")
	r.updateCondition(resource, ConditionPrepared, metav1.ConditionFalse, ReasonPreparing, "Preparing agent")

	// Prepare agent
	output, err := r.BedrockClient.PrepareAgent(ctx, &bedrockagent.PrepareAgentInput{
		AgentId: resource.Status.AgentId,
	})
	if err != nil {
		errMsg := fmt.Sprintf("Failed to prepare agent: %v", err)
		log.Error(err, "Failed to prepare agent")
		r.updateCondition(resource, ConditionPrepared, metav1.ConditionFalse, ReasonAWSError, errMsg)
		r.Recorder.Event(resource, "Warning", "PrepareFailed", errMsg)
		return ReconcileResult{Error: err, RequeueAfter: 30 * time.Second}
	}

	// Update status with preparation info
	resource.Status.Phase = PhasePrepared
	agentStatus := string(output.AgentStatus)
	resource.Status.AgentStatus = &agentStatus
	resource.Status.AgentVersion = output.AgentVersion

	if output.PreparedAt != nil {
		preparedAt := metav1.NewTime(*output.PreparedAt)
		resource.Status.PreparedAt = &preparedAt
	}

	r.updateCondition(resource, ConditionPrepared, metav1.ConditionTrue, ReasonPrepared, "Agent prepared successfully")
	r.updateCondition(resource, ConditionReady, metav1.ConditionTrue, ReasonPrepared, "Agent prepared and ready")
	r.Recorder.Event(resource, "Normal", "Prepared", "Agent prepared successfully")

	log.Info("Agent prepared successfully")
	return ReconcileResult{}
}

// handleDeletion handles the deletion of a BedrockResource
func (r *Reconciler) handleDeletion(ctx context.Context, resource *BedrockResource) ReconcileResult {
	log := klog.FromContext(ctx).WithValues("action", "delete")
	log.Info("Handling resource deletion")

	if !containsFinalizer(resource, BedrockResourceFinalizer) {
		// Finalizer already removed, nothing to do
		return ReconcileResult{}
	}

	resource.Status.Phase = PhaseDeleting
	r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonDeleting, "Deleting resource")

	// Delete the AWS resource based on type
	switch resource.Spec.Type {
	case "agent":
		if resource.Status.AgentId != nil {
			err := r.deleteAgent(ctx, resource)
			if err != nil {
				return ReconcileResult{Error: err, RequeueAfter: 30 * time.Second}
			}
		}
	case "inferenceProfile":
		// TODO: Implement inference profile deletion
		log.Info("Inference profile deletion not yet implemented")
	case "knowledgeBase":
		// TODO: Implement knowledge base deletion
		log.Info("Knowledge base deletion not yet implemented")
	}

	// Remove finalizer and update the resource
	resource.Finalizers = removeFinalizer(resource.Finalizers, BedrockResourceFinalizer)

	// Only update if we have a RestClient (skip in unit tests)
	if r.RestClient != nil {
		if err := r.updateResource(ctx, resource); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ReconcileResult{Error: err, RequeueAfter: time.Second * 10}
		}
	}

	r.Recorder.Event(resource, "Normal", "Deleted", "Resource deleted successfully")

	return ReconcileResult{}
}

// deleteAgent deletes a Bedrock Agent
func (r *Reconciler) deleteAgent(ctx context.Context, resource *BedrockResource) error {
	log := klog.FromContext(ctx).WithValues("agentId", *resource.Status.AgentId)
	log.Info("Deleting agent")

	_, err := r.BedrockClient.DeleteAgent(ctx, &bedrockagent.DeleteAgentInput{
		AgentId: resource.Status.AgentId,
	})
	if err != nil {
		// Check if the agent is already deleted (404 error)
		if isResourceNotFoundError(err) {
			log.Info("Agent already deleted (not found in AWS)")
			return nil // Treat as successful deletion
		}

		errMsg := fmt.Sprintf("Failed to delete agent: %v", err)
		log.Error(err, "Failed to delete agent")
		r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonAWSError, errMsg)
		r.Recorder.Event(resource, "Warning", "DeleteFailed", errMsg)
		return err
	}

	log.Info("Agent deleted successfully")
	return nil
}

// reconcileInferenceProfile handles InferenceProfile reconciliation (placeholder)
func (r *Reconciler) reconcileInferenceProfile(ctx context.Context, resource *BedrockResource) ReconcileResult {
	log := klog.FromContext(ctx).WithValues("type", "inferenceProfile")
	log.Info("Inference profile reconciliation not yet implemented")

	err := fmt.Errorf("inference profile reconciliation not yet implemented")
	r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonReconcileError, err.Error())
	return ReconcileResult{Error: err, RequeueAfter: 5 * time.Minute}
}

// reconcileKnowledgeBase handles KnowledgeBase reconciliation (placeholder)
func (r *Reconciler) reconcileKnowledgeBase(ctx context.Context, resource *BedrockResource) ReconcileResult {
	log := klog.FromContext(ctx).WithValues("type", "knowledgeBase")
	log.Info("Knowledge base reconciliation not yet implemented")

	err := fmt.Errorf("knowledge base reconciliation not yet implemented")
	r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonReconcileError, err.Error())
	return ReconcileResult{Error: err, RequeueAfter: 5 * time.Minute}
}

// Helper functions

const BedrockResourceFinalizer = "bedrock.aws.example.com/finalizer"

func containsFinalizer(resource *BedrockResource, finalizer string) bool {
	for _, f := range resource.Finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

func removeFinalizer(finalizers []string, finalizer string) []string {
	var result []string
	for _, f := range finalizers {
		if f != finalizer {
			result = append(result, f)
		}
	}
	return result
}

// updateCondition updates or adds a condition to the resource status
func (r *Reconciler) updateCondition(resource *BedrockResource, conditionType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.NewTime(time.Now())
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}

	// Find existing condition
	for i, existingCondition := range resource.Status.Conditions {
		if existingCondition.Type == conditionType {
			// Update existing condition
			if existingCondition.Status != status {
				condition.LastTransitionTime = now
			} else {
				condition.LastTransitionTime = existingCondition.LastTransitionTime
			}
			resource.Status.Conditions[i] = condition
			return
		}
	}

	// Add new condition
	resource.Status.Conditions = append(resource.Status.Conditions, condition)
}

// AgentNeedsUpdate checks if an agent needs to be updated
func AgentNeedsUpdate(spec *AgentSpec, agent *types.Agent) bool {
	// Compare basic fields
	if spec.Name != *agent.AgentName {
		return true
	}

	if spec.Description != nil && agent.Description != nil {
		if *spec.Description != *agent.Description {
			return true
		}
	} else if spec.Description != nil || agent.Description != nil {
		return true
	}

	if spec.FoundationModel != nil && agent.FoundationModel != nil {
		if *spec.FoundationModel != *agent.FoundationModel {
			return true
		}
	} else if spec.FoundationModel != nil || agent.FoundationModel != nil {
		return true
	}

	if spec.Instruction != nil && agent.Instruction != nil {
		if *spec.Instruction != *agent.Instruction {
			return true
		}
	} else if spec.Instruction != nil || agent.Instruction != nil {
		return true
	}

	// TODO: Add more detailed comparison logic for other fields

	return false
}

// UpdateResourceStatus updates the status of a BedrockResource in Kubernetes
func (r *Reconciler) UpdateResourceStatus(ctx context.Context, resource *BedrockResource) error {
	// Create a copy of the resource with only the required fields for status update
	statusUpdate := &BedrockResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "bedrock.aws.example.com/v1",
			Kind:       "BedrockResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            resource.Name,
			Namespace:       resource.Namespace,
			ResourceVersion: resource.ResourceVersion,
		},
		Status: resource.Status,
	}

	result := r.RestClient.Put().
		Namespace(resource.Namespace).
		Resource("bedrockresources").
		Name(resource.Name).
		SubResource("status").
		Body(statusUpdate).
		Do(ctx)

	return result.Error()
}

// isResourceNotFoundError checks if the error is a ResourceNotFoundException from AWS
func isResourceNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check for AWS ResourceNotFoundException
	return strings.Contains(err.Error(), "ResourceNotFoundException") ||
		strings.Contains(err.Error(), "StatusCode: 404")
}

// updateResource updates the resource's metadata/spec (not status)
func (r *Reconciler) updateResource(ctx context.Context, resource *BedrockResource) error {
	// Create a copy for the update
	updateResource := &BedrockResource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "bedrock.aws.example.com/v1",
			Kind:       "BedrockResource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            resource.Name,
			Namespace:       resource.Namespace,
			ResourceVersion: resource.ResourceVersion,
			Finalizers:      resource.Finalizers,
		},
		Spec: resource.Spec,
	}

	result := r.RestClient.Put().
		Namespace(resource.Namespace).
		Resource("bedrockresources").
		Name(resource.Name).
		Body(updateResource).
		Do(ctx)

	return result.Error()
}
