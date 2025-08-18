package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
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
		// Update the resource to persist the finalizer (skip in unit tests)
		if r.RestClient != nil {
			if err := r.updateResource(ctx, resource); err != nil {
				log.Error(err, "Failed to add finalizer")
				return ReconcileResult{Error: err, RequeueAfter: time.Second * 10}
			}
			return ReconcileResult{Requeue: true}
		}
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
		if resource.Status.InferenceProfileId != nil {
			err := r.deleteInferenceProfile(ctx, resource)
			if err != nil {
				return ReconcileResult{Error: err, RequeueAfter: 30 * time.Second}
			}
		}
	case "knowledgeBase":
		if resource.Status.KnowledgeBaseId != nil {
			err := r.deleteKnowledgeBase(ctx, resource)
			if err != nil {
				return ReconcileResult{Error: err, RequeueAfter: 30 * time.Second}
			}
		}
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

// deleteInferenceProfile deletes a Bedrock Inference Profile
func (r *Reconciler) deleteInferenceProfile(ctx context.Context, resource *BedrockResource) error {
	log := klog.FromContext(ctx).WithValues("inferenceProfileId", *resource.Status.InferenceProfileId)
	log.Info("Deleting inference profile")

	_, err := r.BedrockClient.DeleteInferenceProfile(ctx, &bedrock.DeleteInferenceProfileInput{
		InferenceProfileIdentifier: resource.Status.InferenceProfileId,
	})
	if err != nil {
		// Check if the profile is already deleted (404 error)
		if isResourceNotFoundError(err) {
			log.Info("Inference profile already deleted (not found in AWS)")
			return nil // Treat as successful deletion
		}

		errMsg := fmt.Sprintf("Failed to delete inference profile: %v", err)
		log.Error(err, "Failed to delete inference profile")
		r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonAWSError, errMsg)
		r.Recorder.Event(resource, "Warning", "DeleteFailed", errMsg)
		return err
	}

	log.Info("Inference profile deleted successfully")
	return nil
}

// reconcileInferenceProfile handles InferenceProfile reconciliation (placeholder)
func (r *Reconciler) reconcileInferenceProfile(ctx context.Context, resource *BedrockResource) ReconcileResult {
	log := klog.FromContext(ctx).WithValues("type", "inferenceProfile")

	if resource.Spec.InferenceProfile == nil {
		err := fmt.Errorf("inferenceProfile spec is required when type is inferenceProfile")
		r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonValidationError, err.Error())
		r.Recorder.Event(resource, "Warning", "ValidationError", err.Error())
		return ReconcileResult{Error: err}
	}

	// Check if inference profile exists in AWS
	var existingProfile *bedrocktypes.InferenceProfileSummary

	// Strategy 1: Check by ID if we have it in status
	if resource.Status.InferenceProfileId != nil {
		log.Info("Found inference profile ID in status", "inferenceProfileId", *resource.Status.InferenceProfileId)
		getOutput, err := r.BedrockClient.GetInferenceProfile(ctx, &bedrock.GetInferenceProfileInput{
			InferenceProfileIdentifier: resource.Status.InferenceProfileId,
		})
		if err != nil {
			if !isResourceNotFoundError(err) {
				errMsg := fmt.Sprintf("Failed to get inference profile: %v", err)
				log.Error(err, "Failed to get inference profile")
				r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonAWSError, errMsg)
				return ReconcileResult{Error: err, RequeueAfter: 30 * time.Second}
			}
			// Profile not found in AWS, will create new one
			log.Info("Inference profile not found in AWS by ID, will search by name or create new one")
		} else {
			existingProfile = &bedrocktypes.InferenceProfileSummary{
				InferenceProfileId:  getOutput.InferenceProfileId,
				InferenceProfileArn: getOutput.InferenceProfileArn,
				Status:              bedrocktypes.InferenceProfileStatusActive, // Assume active if we can get it
			}
		}
	}

	// Strategy 2: If no profile found by ID or status is empty, search by name
	if existingProfile == nil {
		log.Info("No profile ID in status, searching existing profiles by name", "targetName", resource.Spec.InferenceProfile.Name)
		existingProfile = r.findInferenceProfileByName(ctx, resource.Spec.InferenceProfile.Name)
		if existingProfile != nil {
			log.Info("Found existing profile by name discovery", "inferenceProfileId", *existingProfile.InferenceProfileId, "name", resource.Spec.InferenceProfile.Name)
		}
	}

	if existingProfile == nil {
		// Create new inference profile
		log.Info("No existing profile found, will create new one")
		return r.createInferenceProfile(ctx, resource)
	} else {
		log.Info("Found existing profile, updating status", "profileId", *existingProfile.InferenceProfileId)
		// Profile exists, update status and mark as ready
		profileStatus := convertAWSInferenceProfileToStatus(existingProfile)
		resource.Status.InferenceProfileId = profileStatus.InferenceProfileId
		resource.Status.InferenceProfileArn = profileStatus.InferenceProfileArn
		resource.Status.InferenceProfileStatus = profileStatus.InferenceProfileStatus
		resource.Status.Phase = profileStatus.Phase
		resource.Status.ResourceId = profileStatus.ResourceId
		resource.Status.ResourceArn = profileStatus.ResourceArn

		r.updateCondition(resource, ConditionSynced, metav1.ConditionTrue, ReasonReconciling, "Inference profile is up to date")
		r.updateCondition(resource, ConditionReady, metav1.ConditionTrue, ReasonReconciling, "Inference profile is ready")
		return ReconcileResult{}
	}
}

// createInferenceProfile creates a new AWS Inference Profile
func (r *Reconciler) createInferenceProfile(ctx context.Context, resource *BedrockResource) ReconcileResult {
	log := klog.FromContext(ctx).WithValues("action", "create")
	log.Info("Creating new inference profile")

	// Update status to Creating
	resource.Status.Phase = PhaseCreating
	r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonCreating, "Creating inference profile in AWS")
	r.updateCondition(resource, ConditionSynced, metav1.ConditionFalse, ReasonCreating, "Creating inference profile")

	// Convert spec to AWS input
	input := convertInferenceProfileSpecToCreateInput(resource.Spec.InferenceProfile)

	// Create inference profile
	output, err := r.BedrockClient.CreateInferenceProfile(ctx, input)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create inference profile: %v", err)
		log.Error(err, "Failed to create inference profile")
		resource.Status.Phase = PhaseFailed
		r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonAWSError, errMsg)
		r.updateCondition(resource, ConditionSynced, metav1.ConditionFalse, ReasonAWSError, errMsg)
		r.Recorder.Event(resource, "Warning", "CreateFailed", errMsg)
		return ReconcileResult{Error: err, RequeueAfter: 30 * time.Second}
	}

	// Update status with created profile info
	// Extract ID from ARN (format: arn:aws:bedrock:region:account:inference-profile/profile-id)
	arn := *output.InferenceProfileArn
	profileId := arn[strings.LastIndex(arn, "/")+1:]
	resource.Status.InferenceProfileId = aws.String(profileId)
	resource.Status.InferenceProfileArn = output.InferenceProfileArn
	resource.Status.ResourceId = resource.Status.InferenceProfileId
	resource.Status.ResourceArn = output.InferenceProfileArn
	resource.Status.Phase = PhasePrepared // Inference profiles are immediately ready

	if output.Status != "" {
		status := string(output.Status)
		resource.Status.InferenceProfileStatus = &status
	}

	r.updateCondition(resource, ConditionSynced, metav1.ConditionTrue, ReasonCreated, "Inference profile created successfully")
	r.updateCondition(resource, ConditionReady, metav1.ConditionTrue, ReasonCreated, "Inference profile created and ready")
	r.Recorder.Event(resource, "Normal", "Created", fmt.Sprintf("Inference profile created with ID %s", *resource.Status.InferenceProfileId))

	log.Info("Inference profile created successfully", "inferenceProfileId", *resource.Status.InferenceProfileId)
	return ReconcileResult{}
}

// findInferenceProfileByName searches for an existing inference profile by name
// This is used as a fallback when the status doesn't contain the profile ID
func (r *Reconciler) findInferenceProfileByName(ctx context.Context, profileName string) *bedrocktypes.InferenceProfileSummary {
	log := klog.FromContext(ctx).WithValues("action", "findByName", "profileName", profileName)

	// List all application inference profiles
	listOutput, err := r.BedrockClient.ListInferenceProfiles(ctx, &bedrock.ListInferenceProfilesInput{
		TypeEquals: bedrocktypes.InferenceProfileTypeApplication,
	})
	if err != nil {
		log.Error(err, "Failed to list inference profiles for name search")
		return nil
	}

	// Search for a profile with matching name
	for _, profile := range listOutput.InferenceProfileSummaries {
		if profile.InferenceProfileName != nil && *profile.InferenceProfileName == profileName {
			log.V(1).Info("Found matching profile by name", "inferenceProfileId", *profile.InferenceProfileId)
			return &profile
		}
	}

	log.V(1).Info("No existing profile found with matching name")
	return nil
}

// reconcileKnowledgeBase handles KnowledgeBase reconciliation
func (r *Reconciler) reconcileKnowledgeBase(ctx context.Context, resource *BedrockResource) ReconcileResult {
	log := klog.FromContext(ctx).WithValues("type", "knowledgeBase")

	if resource.Spec.KnowledgeBase == nil {
		err := fmt.Errorf("knowledgeBase spec is required when type is knowledgeBase")
		r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonValidationError, err.Error())
		return ReconcileResult{Error: err}
	}

	// Check if knowledge base already exists
	var existingKB *types.KnowledgeBase

	// First, try to find by ID if we have it in status
	if resource.Status.KnowledgeBaseId != nil {
		log.V(1).Info("Looking up knowledge base by ID", "knowledgeBaseId", *resource.Status.KnowledgeBaseId)

		getOutput, err := r.BedrockClient.GetKnowledgeBase(ctx, &bedrockagent.GetKnowledgeBaseInput{
			KnowledgeBaseId: resource.Status.KnowledgeBaseId,
		})
		if err != nil {
			if !isResourceNotFoundError(err) {
				r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonReconcileError,
					fmt.Sprintf("Failed to get knowledge base: %v", err))
				return ReconcileResult{Error: err, RequeueAfter: time.Minute}
			}
			log.V(1).Info("Knowledge base not found by ID, will search by name or create new one")
		} else {
			existingKB = getOutput.KnowledgeBase
			log.V(1).Info("Found existing knowledge base by ID", "status", existingKB.Status)
		}
	}

	// If not found by ID, try to find by name (stale cache scenario)
	if existingKB == nil {
		foundKB := r.findKnowledgeBaseByName(ctx, resource.Spec.KnowledgeBase.Name)
		if foundKB != nil {
			existingKB = foundKB
			log.V(1).Info("Found existing knowledge base by name", "status", existingKB.Status)
		}
	}

	if existingKB == nil {
		// Create new knowledge base
		log.Info("Creating new knowledge base", "name", resource.Spec.KnowledgeBase.Name)

		result := r.createKnowledgeBase(ctx, resource)
		if result.Error != nil {
			return result
		}

		r.updateCondition(resource, ConditionReady, metav1.ConditionTrue, ReasonReconcileSuccess, "Knowledge base created successfully")
		r.Recorder.Event(resource, "Normal", "Created", "Knowledge base created successfully")
		return result
	}

	// Update existing knowledge base if needed
	log.V(1).Info("Knowledge base already exists, checking for updates", "status", existingKB.Status)

	// Update status from existing knowledge base
	status := ConvertAWSKnowledgeBaseToStatus(existingKB)
	mergeStatus(&resource.Status, status)

	// Check if Knowledge Base is ACTIVE and we need to start pending ingestion jobs
	if existingKB.Status == types.KnowledgeBaseStatusActive {
		if err := r.checkAndStartPendingIngestionJobs(ctx, resource, *existingKB.KnowledgeBaseId); err != nil {
			log.Error(err, "Failed to start pending ingestion jobs", "knowledgeBaseId", *existingKB.KnowledgeBaseId)
		}
	}

	// Check if update is needed (simplified - just description for now)
	needsUpdate := false
	if resource.Spec.KnowledgeBase.Description != nil && existingKB.Description != nil {
		if *resource.Spec.KnowledgeBase.Description != *existingKB.Description {
			needsUpdate = true
		}
	} else if resource.Spec.KnowledgeBase.Description != existingKB.Description {
		needsUpdate = true
	}

	if needsUpdate {
		log.Info("Updating knowledge base", "knowledgeBaseId", *existingKB.KnowledgeBaseId)

		updateInput := &bedrockagent.UpdateKnowledgeBaseInput{
			KnowledgeBaseId: existingKB.KnowledgeBaseId,
			Name:            aws.String(resource.Spec.KnowledgeBase.Name),
			RoleArn:         aws.String(resource.Spec.KnowledgeBase.RoleArn),
		}

		if resource.Spec.KnowledgeBase.Description != nil {
			updateInput.Description = resource.Spec.KnowledgeBase.Description
		}

		_, err := r.BedrockClient.UpdateKnowledgeBase(ctx, updateInput)
		if err != nil {
			r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonReconcileError,
				fmt.Sprintf("Failed to update knowledge base: %v", err))
			return ReconcileResult{Error: err, RequeueAfter: time.Minute}
		}

		r.Recorder.Event(resource, "Normal", "Updated", "Knowledge base updated successfully")
	}

	r.updateCondition(resource, ConditionReady, metav1.ConditionTrue, ReasonReconcileSuccess, "Knowledge base reconciled successfully")
	return ReconcileResult{}
}

// createKnowledgeBase creates a new AWS Bedrock Knowledge Base
func (r *Reconciler) createKnowledgeBase(ctx context.Context, resource *BedrockResource) ReconcileResult {
	log := klog.FromContext(ctx)

	// Convert spec to AWS input
	input := ConvertKnowledgeBaseSpecToCreateInput(resource.Spec.KnowledgeBase)

	log.Info("Creating knowledge base in AWS", "name", *input.Name)

	// First, ensure the OpenSearch index exists before creating the Knowledge Base
	if err := r.ensureOpenSearchIndex(ctx, resource); err != nil {
		resource.Status.Phase = PhaseFailed
		r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonReconcileError,
			fmt.Sprintf("Failed to create OpenSearch index: %v", err))
		return ReconcileResult{Error: err, RequeueAfter: time.Minute}
	}

	output, err := r.BedrockClient.CreateKnowledgeBase(ctx, input)
	if err != nil {
		resource.Status.Phase = PhaseFailed
		r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonReconcileError,
			fmt.Sprintf("Failed to create knowledge base: %v", err))
		return ReconcileResult{Error: err, RequeueAfter: time.Minute}
	}

	// Update status with new knowledge base information
	status := ConvertAWSKnowledgeBaseToStatus(output.KnowledgeBase)
	mergeStatus(&resource.Status, status)

	log.Info("Knowledge base created successfully", "knowledgeBaseId", *output.KnowledgeBase.KnowledgeBaseId)

	// Create data sources if specified
	if len(resource.Spec.KnowledgeBase.DataSources) > 0 {
		if err := r.createDataSources(ctx, resource, *output.KnowledgeBase.KnowledgeBaseId); err != nil {
			log.Error(err, "Failed to create data sources", "knowledgeBaseId", *output.KnowledgeBase.KnowledgeBaseId)
			// Don't fail the entire operation, data sources can be created later
			r.updateCondition(resource, ConditionReady, metav1.ConditionFalse, ReasonReconcileError,
				fmt.Sprintf("Knowledge base created but failed to create data sources: %v", err))
			return ReconcileResult{Error: err, RequeueAfter: time.Minute}
		}
	}

	return ReconcileResult{}
}

// deleteKnowledgeBase deletes an AWS Bedrock Knowledge Base
func (r *Reconciler) deleteKnowledgeBase(ctx context.Context, resource *BedrockResource) error {
	log := klog.FromContext(ctx)

	if resource.Status.KnowledgeBaseId == nil {
		log.Info("No knowledge base ID found in status, considering deletion complete")
		return nil
	}

	log.Info("Deleting knowledge base", "knowledgeBaseId", *resource.Status.KnowledgeBaseId)

	_, err := r.BedrockClient.DeleteKnowledgeBase(ctx, &bedrockagent.DeleteKnowledgeBaseInput{
		KnowledgeBaseId: resource.Status.KnowledgeBaseId,
	})

	if err != nil {
		if isResourceNotFoundError(err) {
			log.Info("Knowledge base already deleted or not found")
			return nil
		}
		return err
	}

	log.Info("Knowledge base deleted successfully")
	return nil
}

// findKnowledgeBaseByName searches for an existing knowledge base by name
func (r *Reconciler) findKnowledgeBaseByName(ctx context.Context, name string) *types.KnowledgeBase {
	log := klog.FromContext(ctx)

	log.V(1).Info("Searching for knowledge base by name", "name", name)

	// List all knowledge bases and find the one with matching name
	listOutput, err := r.BedrockClient.ListKnowledgeBases(ctx, &bedrockagent.ListKnowledgeBasesInput{})
	if err != nil {
		log.Error(err, "Failed to list knowledge bases during name search")
		return nil
	}

	for _, kbSummary := range listOutput.KnowledgeBaseSummaries {
		if kbSummary.Name != nil && *kbSummary.Name == name {
			log.V(1).Info("Found matching knowledge base by name", "knowledgeBaseId", *kbSummary.KnowledgeBaseId)

			// Get the full knowledge base details
			getOutput, err := r.BedrockClient.GetKnowledgeBase(ctx, &bedrockagent.GetKnowledgeBaseInput{
				KnowledgeBaseId: kbSummary.KnowledgeBaseId,
			})
			if err != nil {
				log.Error(err, "Failed to get knowledge base details", "knowledgeBaseId", *kbSummary.KnowledgeBaseId)
				continue
			}

			return getOutput.KnowledgeBase
		}
	}

	log.V(1).Info("No existing knowledge base found with matching name")
	return nil
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
// with retry logic for optimistic concurrency control
func (r *Reconciler) UpdateResourceStatus(ctx context.Context, resource *BedrockResource) error {
	log := klog.FromContext(ctx).WithValues("action", "updateStatus")

	// Retry up to 3 times for optimistic concurrency conflicts
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			log.V(1).Info("Retrying status update", "attempt", attempt+1)
			// Get the latest resource version from the server
			result := r.RestClient.Get().
				Namespace(resource.Namespace).
				Resource("bedrockresources").
				Name(resource.Name).
				Do(ctx)

			if err := result.Error(); err != nil {
				log.Error(err, "Failed to get latest resource for retry")
				return err
			}

			var latestResource BedrockResource
			if err := result.Into(&latestResource); err != nil {
				log.Error(err, "Failed to decode latest resource")
				return err
			}

			// Update the resource version and preserve our status changes
			resource.ResourceVersion = latestResource.ResourceVersion
			resource.Generation = latestResource.Generation
		}

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

		err := result.Error()
		if err == nil {
			log.V(1).Info("Successfully updated resource status")
			return nil
		}

		// Check if it's a conflict error (409)
		if strings.Contains(err.Error(), "the object has been modified") {
			log.V(1).Info("Status update conflict, will retry", "attempt", attempt+1, "error", err)
			continue
		}

		// Non-conflict error, return immediately
		log.Error(err, "Failed to update resource status")
		return err
	}

	return fmt.Errorf("failed to update status after 3 attempts due to conflicts")
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

// mergeStatus merges status fields from source into target, preserving existing non-nil fields
func mergeStatus(target *BedrockResourceStatus, source *BedrockResourceStatus) {
	if source.Phase != "" {
		target.Phase = source.Phase
	}

	// Agent-specific fields
	if source.AgentId != nil {
		target.AgentId = source.AgentId
	}
	if source.AgentArn != nil {
		target.AgentArn = source.AgentArn
	}
	if source.AgentStatus != nil {
		target.AgentStatus = source.AgentStatus
	}
	if source.AgentVersion != nil {
		target.AgentVersion = source.AgentVersion
	}
	if source.PreparedAt != nil {
		target.PreparedAt = source.PreparedAt
	}

	// InferenceProfile-specific fields
	if source.InferenceProfileId != nil {
		target.InferenceProfileId = source.InferenceProfileId
	}
	if source.InferenceProfileArn != nil {
		target.InferenceProfileArn = source.InferenceProfileArn
	}
	if source.InferenceProfileStatus != nil {
		target.InferenceProfileStatus = source.InferenceProfileStatus
	}
	if source.InferenceProfileType != nil {
		target.InferenceProfileType = source.InferenceProfileType
	}

	// KnowledgeBase-specific fields
	if source.KnowledgeBaseId != nil {
		target.KnowledgeBaseId = source.KnowledgeBaseId
	}
	if source.KnowledgeBaseArn != nil {
		target.KnowledgeBaseArn = source.KnowledgeBaseArn
	}
	if source.KnowledgeBaseStatus != nil {
		target.KnowledgeBaseStatus = source.KnowledgeBaseStatus
	}
	if len(source.DataSourceIds) > 0 {
		target.DataSourceIds = make([]string, len(source.DataSourceIds))
		copy(target.DataSourceIds, source.DataSourceIds)
	}
	if len(source.FailureReasons) > 0 {
		target.FailureReasons = make([]string, len(source.FailureReasons))
		copy(target.FailureReasons, source.FailureReasons)
	}

	// Generic fields
	if source.ResourceId != nil {
		target.ResourceId = source.ResourceId
	}
	if source.ResourceArn != nil {
		target.ResourceArn = source.ResourceArn
	}
	if source.LastSyncTime != nil {
		target.LastSyncTime = source.LastSyncTime
	}
}

// ensureOpenSearchIndex ensures the OpenSearch index exists before creating a Knowledge Base
func (r *Reconciler) ensureOpenSearchIndex(ctx context.Context, resource *BedrockResource) error {
	log := klog.FromContext(ctx)

	kb := resource.Spec.KnowledgeBase
	if kb == nil || kb.OpenSearchServerlessConfiguration == nil {
		return fmt.Errorf("invalid Knowledge Base configuration: missing OpenSearch configuration")
	}

	opensearchConfig := kb.OpenSearchServerlessConfiguration

	// Extract collection ID from ARN
	// ARN format: arn:aws:aoss:region:account:collection/collection-id
	collectionId, err := extractCollectionIdFromArn(opensearchConfig.CollectionArn)
	if err != nil {
		return fmt.Errorf("failed to extract collection ID: %w", err)
	}

	indexName := opensearchConfig.VectorIndexName
	// Default vector dimension for Bedrock (Claude/Titan embeddings typically use 1536)
	vectorDimension := 1536

	log.Info("Ensuring OpenSearch index exists", "collectionId", collectionId, "index", indexName)

	// Check if index already exists
	exists, err := r.BedrockClient.CheckOpenSearchIndexExists(ctx, collectionId, indexName)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}

	if exists {
		log.Info("OpenSearch index already exists", "index", indexName)
		return nil
	}

	// Create the index
	log.Info("Creating OpenSearch index", "index", indexName, "dimension", vectorDimension)
	err = r.BedrockClient.CreateOpenSearchIndex(ctx, collectionId, indexName, vectorDimension)
	if err != nil {
		return fmt.Errorf("failed to create OpenSearch index: %w", err)
	}

	log.Info("OpenSearch index created successfully", "index", indexName)
	return nil
}

// createDataSources creates all data sources for a knowledge base
func (r *Reconciler) createDataSources(ctx context.Context, resource *BedrockResource, knowledgeBaseId string) error {
	log := klog.FromContext(ctx)

	if resource.Spec.KnowledgeBase == nil || len(resource.Spec.KnowledgeBase.DataSources) == 0 {
		return nil
	}

	var dataSourceStatuses []DataSourceStatus
	var dataSourceIds []string

	for _, dsSpec := range resource.Spec.KnowledgeBase.DataSources {
		log.Info("Creating data source", "name", dsSpec.Name, "type", dsSpec.DataSourceType, "knowledgeBaseId", knowledgeBaseId)

		// Create data source in AWS
		dsOutput, err := r.createSingleDataSource(ctx, knowledgeBaseId, dsSpec)
		if err != nil {
			log.Error(err, "Failed to create data source", "name", dsSpec.Name)
			// Add failed data source to status
			dataSourceStatuses = append(dataSourceStatuses, DataSourceStatus{
				Name:           dsSpec.Name,
				DataSourceId:   "",
				Status:         "FAILED",
				FailureReasons: []string{err.Error()},
			})
			continue
		}

		// Track successful data source
		dataSourceId := *dsOutput.DataSource.DataSourceId
		dataSourceIds = append(dataSourceIds, dataSourceId)
		dataSourceStatuses = append(dataSourceStatuses, DataSourceStatus{
			Name:         dsSpec.Name,
			DataSourceId: dataSourceId,
			Status:       string(dsOutput.DataSource.Status),
		})

		log.Info("Data source created successfully", "name", dsSpec.Name, "dataSourceId", dataSourceId)

		// Start ingestion job for the data source (only if Knowledge Base is ready)
		if err := r.startIngestionJob(ctx, knowledgeBaseId, dataSourceId, dsSpec.Name); err != nil {
			// Check if it's a "Knowledge Base not ready" error
			if strings.Contains(err.Error(), "CREATING") || strings.Contains(err.Error(), "ConflictException") {
				log.Info("Knowledge Base not ready for ingestion yet, will retry later", "dataSourceId", dataSourceId)
				// Don't mark as failure, just note that ingestion is pending
			} else {
				log.Error(err, "Failed to start ingestion job", "dataSourceId", dataSourceId)
				// Update status with ingestion failure for real errors
				for i := range dataSourceStatuses {
					if dataSourceStatuses[i].DataSourceId == dataSourceId {
						dataSourceStatuses[i].FailureReasons = append(dataSourceStatuses[i].FailureReasons,
							fmt.Sprintf("Failed to start ingestion: %v", err))
						break
					}
				}
			}
		}
	}

	// Update resource status with data source information
	resource.Status.DataSourceIds = dataSourceIds
	resource.Status.DataSources = dataSourceStatuses

	log.Info("Data source creation completed", "totalDataSources", len(resource.Spec.KnowledgeBase.DataSources),
		"successful", len(dataSourceIds), "failed", len(dataSourceStatuses)-len(dataSourceIds))

	return nil
}

// extractCollectionIdFromArn extracts the OpenSearch Serverless collection ID from ARN
func extractCollectionIdFromArn(collectionArn string) (string, error) {
	// ARN format: arn:aws:aoss:region:account:collection/collection-id

	parts := strings.Split(collectionArn, ":")
	if len(parts) != 6 || parts[0] != "arn" || parts[1] != "aws" || parts[2] != "aoss" {
		return "", fmt.Errorf("invalid OpenSearch Serverless collection ARN format: %s", collectionArn)
	}

	collectionPart := parts[5] // "collection/collection-id"

	collectionParts := strings.Split(collectionPart, "/")
	if len(collectionParts) != 2 || collectionParts[0] != "collection" {
		return "", fmt.Errorf("invalid collection part in ARN: %s", collectionPart)
	}

	collectionId := collectionParts[1]

	return collectionId, nil
}

// createSingleDataSource creates a single data source for a knowledge base
func (r *Reconciler) createSingleDataSource(ctx context.Context, knowledgeBaseId string, dsSpec DataSourceSpec) (*bedrockagent.CreateDataSourceOutput, error) {
	input := &bedrockagent.CreateDataSourceInput{
		KnowledgeBaseId:         &knowledgeBaseId,
		Name:                    &dsSpec.Name,
		DataSourceConfiguration: &types.DataSourceConfiguration{},
	}

	// Set description if provided
	if dsSpec.Description != nil {
		input.Description = dsSpec.Description
	}

	// Configure based on data source type
	switch dsSpec.DataSourceType {
	case "S3":
		if dsSpec.S3Configuration == nil {
			return nil, fmt.Errorf("S3 configuration is required for S3 data source")
		}

		s3Config := &types.S3DataSourceConfiguration{
			BucketArn: &dsSpec.S3Configuration.BucketArn,
		}

		// TODO: Add support for inclusion/exclusion patterns once we verify the AWS SDK structure
		// Note: The AWS SDK structure may have different field names or may require different configuration

		input.DataSourceConfiguration.Type = types.DataSourceTypeS3
		input.DataSourceConfiguration.S3Configuration = s3Config

	default:
		return nil, fmt.Errorf("unsupported data source type: %s", dsSpec.DataSourceType)
	}

	return r.BedrockClient.CreateDataSource(ctx, input)
}

// startIngestionJob starts an ingestion job for a data source
func (r *Reconciler) startIngestionJob(ctx context.Context, knowledgeBaseId, dataSourceId, dataSourceName string) error {
	log := klog.FromContext(ctx)

	input := &bedrockagent.StartIngestionJobInput{
		KnowledgeBaseId: &knowledgeBaseId,
		DataSourceId:    &dataSourceId,
	}

	output, err := r.BedrockClient.StartIngestionJob(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to start ingestion job: %w", err)
	}

	log.Info("Ingestion job started", "dataSourceName", dataSourceName,
		"ingestionJobId", *output.IngestionJob.IngestionJobId,
		"status", string(output.IngestionJob.Status))

	return nil
}

// checkAndStartPendingIngestionJobs checks if there are data sources without recent ingestion jobs and starts them
func (r *Reconciler) checkAndStartPendingIngestionJobs(ctx context.Context, resource *BedrockResource, knowledgeBaseId string) error {
	log := klog.FromContext(ctx)

	if resource.Spec.KnowledgeBase == nil || len(resource.Spec.KnowledgeBase.DataSources) == 0 {
		return nil
	}

	// List existing data sources
	listOutput, err := r.BedrockClient.ListDataSources(ctx, &bedrockagent.ListDataSourcesInput{
		KnowledgeBaseId: &knowledgeBaseId,
	})
	if err != nil {
		return fmt.Errorf("failed to list data sources: %w", err)
	}

	// Check each data source for recent ingestion jobs
	for _, ds := range listOutput.DataSourceSummaries {
		// List ingestion jobs for this data source
		jobsOutput, err := r.BedrockClient.ListIngestionJobs(ctx, &bedrockagent.ListIngestionJobsInput{
			KnowledgeBaseId: &knowledgeBaseId,
			DataSourceId:    ds.DataSourceId,
		})
		if err != nil {
			log.Error(err, "Failed to list ingestion jobs", "dataSourceId", *ds.DataSourceId)
			continue
		}

		// If no ingestion jobs exist, start one
		if len(jobsOutput.IngestionJobSummaries) == 0 {
			log.Info("Starting ingestion job for data source with no previous jobs", "dataSourceId", *ds.DataSourceId, "name", *ds.Name)
			if err := r.startIngestionJob(ctx, knowledgeBaseId, *ds.DataSourceId, *ds.Name); err != nil {
				log.Error(err, "Failed to start ingestion job", "dataSourceId", *ds.DataSourceId)
			}
		}
	}

	return nil
}
