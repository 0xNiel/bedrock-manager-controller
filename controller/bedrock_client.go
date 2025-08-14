package controller

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BedrockClient defines the interface for AWS Bedrock operations
// This interface allows for easy mocking in unit tests
type BedrockClient interface {
	// Agent operations
	CreateAgent(ctx context.Context, params *bedrockagent.CreateAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateAgentOutput, error)
	GetAgent(ctx context.Context, params *bedrockagent.GetAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetAgentOutput, error)
	UpdateAgent(ctx context.Context, params *bedrockagent.UpdateAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateAgentOutput, error)
	DeleteAgent(ctx context.Context, params *bedrockagent.DeleteAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteAgentOutput, error)
	PrepareAgent(ctx context.Context, params *bedrockagent.PrepareAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.PrepareAgentOutput, error)

	// Future: Inference Profile operations will be added here
	// Future: Knowledge Base operations will be added here
}

// bedrockClientImpl wraps the AWS SDK client to implement our interface
type bedrockClientImpl struct {
	client *bedrockagent.Client
}

// NewBedrockClient creates a new Bedrock client wrapper
func NewBedrockClient(client *bedrockagent.Client) BedrockClient {
	return &bedrockClientImpl{
		client: client,
	}
}

// Agent operations implementation
func (b *bedrockClientImpl) CreateAgent(ctx context.Context, params *bedrockagent.CreateAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateAgentOutput, error) {
	return b.client.CreateAgent(ctx, params, optFns...)
}

func (b *bedrockClientImpl) GetAgent(ctx context.Context, params *bedrockagent.GetAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetAgentOutput, error) {
	return b.client.GetAgent(ctx, params, optFns...)
}

func (b *bedrockClientImpl) UpdateAgent(ctx context.Context, params *bedrockagent.UpdateAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateAgentOutput, error) {
	return b.client.UpdateAgent(ctx, params, optFns...)
}

func (b *bedrockClientImpl) DeleteAgent(ctx context.Context, params *bedrockagent.DeleteAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteAgentOutput, error) {
	return b.client.DeleteAgent(ctx, params, optFns...)
}

func (b *bedrockClientImpl) PrepareAgent(ctx context.Context, params *bedrockagent.PrepareAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.PrepareAgentOutput, error) {
	return b.client.PrepareAgent(ctx, params, optFns...)
}

// Helper functions for converting between our types and AWS SDK types

// convertAgentSpecToCreateInput converts our AgentSpec to AWS CreateAgentInput
func convertAgentSpecToCreateInput(spec *AgentSpec) *bedrockagent.CreateAgentInput {
	input := &bedrockagent.CreateAgentInput{
		AgentName: &spec.Name,
	}

	if spec.Description != nil {
		input.Description = spec.Description
	}

	if spec.FoundationModel != nil {
		input.FoundationModel = spec.FoundationModel
	}

	if spec.Instruction != nil {
		input.Instruction = spec.Instruction
	}

	if spec.AgentResourceRoleArn != nil {
		input.AgentResourceRoleArn = spec.AgentResourceRoleArn
	}

	if spec.CustomerEncryptionKeyArn != nil {
		input.CustomerEncryptionKeyArn = spec.CustomerEncryptionKeyArn
	}

	if spec.IdleSessionTTLInSeconds != nil {
		input.IdleSessionTTLInSeconds = spec.IdleSessionTTLInSeconds
	}

	if spec.Tags != nil {
		input.Tags = spec.Tags
	}

	if spec.GuardrailConfiguration != nil {
		input.GuardrailConfiguration = &types.GuardrailConfiguration{
			GuardrailIdentifier: spec.GuardrailConfiguration.GuardrailId,
			GuardrailVersion:    spec.GuardrailConfiguration.GuardrailVersion,
		}
	}

	if spec.MemoryConfiguration != nil {
		memoryConfig := &types.MemoryConfiguration{}

		if len(spec.MemoryConfiguration.EnabledMemoryTypes) > 0 {
			var enabledTypes []types.MemoryType
			for _, memType := range spec.MemoryConfiguration.EnabledMemoryTypes {
				enabledTypes = append(enabledTypes, types.MemoryType(memType))
			}
			memoryConfig.EnabledMemoryTypes = enabledTypes
		}

		if spec.MemoryConfiguration.StorageDays != nil {
			memoryConfig.StorageDays = spec.MemoryConfiguration.StorageDays
		}

		input.MemoryConfiguration = memoryConfig
	}

	return input
}

// convertAgentSpecToUpdateInput converts our AgentSpec to AWS UpdateAgentInput
func convertAgentSpecToUpdateInput(spec *AgentSpec, agentId string) *bedrockagent.UpdateAgentInput {
	input := &bedrockagent.UpdateAgentInput{
		AgentId:   &agentId,
		AgentName: &spec.Name,
	}

	if spec.Description != nil {
		input.Description = spec.Description
	}

	if spec.FoundationModel != nil {
		input.FoundationModel = spec.FoundationModel
	}

	if spec.Instruction != nil {
		input.Instruction = spec.Instruction
	}

	if spec.AgentResourceRoleArn != nil {
		input.AgentResourceRoleArn = spec.AgentResourceRoleArn
	}

	if spec.CustomerEncryptionKeyArn != nil {
		input.CustomerEncryptionKeyArn = spec.CustomerEncryptionKeyArn
	}

	if spec.IdleSessionTTLInSeconds != nil {
		input.IdleSessionTTLInSeconds = spec.IdleSessionTTLInSeconds
	}

	if spec.GuardrailConfiguration != nil {
		input.GuardrailConfiguration = &types.GuardrailConfiguration{
			GuardrailIdentifier: spec.GuardrailConfiguration.GuardrailId,
			GuardrailVersion:    spec.GuardrailConfiguration.GuardrailVersion,
		}
	}

	if spec.MemoryConfiguration != nil {
		memoryConfig := &types.MemoryConfiguration{}

		if len(spec.MemoryConfiguration.EnabledMemoryTypes) > 0 {
			var enabledTypes []types.MemoryType
			for _, memType := range spec.MemoryConfiguration.EnabledMemoryTypes {
				enabledTypes = append(enabledTypes, types.MemoryType(memType))
			}
			memoryConfig.EnabledMemoryTypes = enabledTypes
		}

		if spec.MemoryConfiguration.StorageDays != nil {
			memoryConfig.StorageDays = spec.MemoryConfiguration.StorageDays
		}

		input.MemoryConfiguration = memoryConfig
	}

	return input
}

// convertAWSAgentToStatus converts AWS Agent to our status fields
func convertAWSAgentToStatus(agent *types.Agent) *BedrockResourceStatus {
	status := &BedrockResourceStatus{
		Phase:       mapAgentStatusToPhase(agent.AgentStatus),
		AgentId:     agent.AgentId,
		AgentArn:    agent.AgentArn,
		ResourceId:  agent.AgentId,
		ResourceArn: agent.AgentArn,
	}

	if agent.AgentStatus != "" {
		agentStatus := string(agent.AgentStatus)
		status.AgentStatus = &agentStatus
	}

	if agent.AgentVersion != nil {
		status.AgentVersion = agent.AgentVersion
	}

	if agent.PreparedAt != nil {
		// Convert time.Time to metav1.Time
		preparedAt := metav1.NewTime(*agent.PreparedAt)
		status.PreparedAt = &preparedAt
	}

	return status
}

// mapAgentStatusToPhase maps AWS AgentStatus to our Phase
func mapAgentStatusToPhase(agentStatus types.AgentStatus) string {
	switch agentStatus {
	case types.AgentStatusCreating:
		return PhaseCreating
	case types.AgentStatusNotPrepared:
		return PhaseCreated
	case types.AgentStatusPreparing:
		return PhasePreparing
	case types.AgentStatusPrepared:
		return PhasePrepared
	case types.AgentStatusUpdating:
		return PhaseUpdating
	case types.AgentStatusDeleting:
		return PhaseDeleting
	case types.AgentStatusFailed:
		return PhaseFailed
	default:
		return PhaseCreating
	}
}
