package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	"github.com/aws/aws-sdk-go-v2/service/opensearchserverless"
	"github.com/aws/aws-sdk-go-v2/service/opensearchserverless/document"
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

	// Inference Profile operations
	CreateInferenceProfile(ctx context.Context, params *bedrock.CreateInferenceProfileInput, optFns ...func(*bedrock.Options)) (*bedrock.CreateInferenceProfileOutput, error)
	GetInferenceProfile(ctx context.Context, params *bedrock.GetInferenceProfileInput, optFns ...func(*bedrock.Options)) (*bedrock.GetInferenceProfileOutput, error)
	DeleteInferenceProfile(ctx context.Context, params *bedrock.DeleteInferenceProfileInput, optFns ...func(*bedrock.Options)) (*bedrock.DeleteInferenceProfileOutput, error)
	ListInferenceProfiles(ctx context.Context, params *bedrock.ListInferenceProfilesInput, optFns ...func(*bedrock.Options)) (*bedrock.ListInferenceProfilesOutput, error)

	// Knowledge Base operations
	CreateKnowledgeBase(ctx context.Context, params *bedrockagent.CreateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateKnowledgeBaseOutput, error)
	GetKnowledgeBase(ctx context.Context, params *bedrockagent.GetKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetKnowledgeBaseOutput, error)
	UpdateKnowledgeBase(ctx context.Context, params *bedrockagent.UpdateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateKnowledgeBaseOutput, error)
	DeleteKnowledgeBase(ctx context.Context, params *bedrockagent.DeleteKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteKnowledgeBaseOutput, error)
	ListKnowledgeBases(ctx context.Context, params *bedrockagent.ListKnowledgeBasesInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.ListKnowledgeBasesOutput, error)

	// Data Source operations (for Knowledge Bases)
	CreateDataSource(ctx context.Context, params *bedrockagent.CreateDataSourceInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateDataSourceOutput, error)
	GetDataSource(ctx context.Context, params *bedrockagent.GetDataSourceInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetDataSourceOutput, error)
	UpdateDataSource(ctx context.Context, params *bedrockagent.UpdateDataSourceInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateDataSourceOutput, error)
	DeleteDataSource(ctx context.Context, params *bedrockagent.DeleteDataSourceInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteDataSourceOutput, error)
	ListDataSources(ctx context.Context, params *bedrockagent.ListDataSourcesInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.ListDataSourcesOutput, error)
	StartIngestionJob(ctx context.Context, params *bedrockagent.StartIngestionJobInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.StartIngestionJobOutput, error)
	ListIngestionJobs(ctx context.Context, params *bedrockagent.ListIngestionJobsInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.ListIngestionJobsOutput, error)

	// OpenSearch operations
	CreateOpenSearchIndex(ctx context.Context, collectionId, indexName string, vectorDimension int) error
	CheckOpenSearchIndexExists(ctx context.Context, collectionId, indexName string) (bool, error)
}

// bedrockClientImpl wraps the AWS SDK clients to implement our interface
type bedrockClientImpl struct {
	agentClient                *bedrockagent.Client
	bedrockClient              *bedrock.Client
	opensearchServerlessClient *opensearchserverless.Client
	awsConfig                  aws.Config
}

// NewBedrockClient creates a new Bedrock client wrapper
func NewBedrockClient(agentClient *bedrockagent.Client, bedrockClient *bedrock.Client, awsConfig aws.Config) BedrockClient {
	return &bedrockClientImpl{
		agentClient:                agentClient,
		bedrockClient:              bedrockClient,
		opensearchServerlessClient: opensearchserverless.NewFromConfig(awsConfig),
		awsConfig:                  awsConfig,
	}
}

// Agent operations implementation
func (b *bedrockClientImpl) CreateAgent(ctx context.Context, params *bedrockagent.CreateAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateAgentOutput, error) {
	return b.agentClient.CreateAgent(ctx, params, optFns...)
}

func (b *bedrockClientImpl) GetAgent(ctx context.Context, params *bedrockagent.GetAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetAgentOutput, error) {
	return b.agentClient.GetAgent(ctx, params, optFns...)
}

func (b *bedrockClientImpl) UpdateAgent(ctx context.Context, params *bedrockagent.UpdateAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateAgentOutput, error) {
	return b.agentClient.UpdateAgent(ctx, params, optFns...)
}

func (b *bedrockClientImpl) DeleteAgent(ctx context.Context, params *bedrockagent.DeleteAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteAgentOutput, error) {
	return b.agentClient.DeleteAgent(ctx, params, optFns...)
}

func (b *bedrockClientImpl) PrepareAgent(ctx context.Context, params *bedrockagent.PrepareAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.PrepareAgentOutput, error) {
	return b.agentClient.PrepareAgent(ctx, params, optFns...)
}

// Inference Profile operations implementation
func (b *bedrockClientImpl) CreateInferenceProfile(ctx context.Context, params *bedrock.CreateInferenceProfileInput, optFns ...func(*bedrock.Options)) (*bedrock.CreateInferenceProfileOutput, error) {
	return b.bedrockClient.CreateInferenceProfile(ctx, params, optFns...)
}

func (b *bedrockClientImpl) GetInferenceProfile(ctx context.Context, params *bedrock.GetInferenceProfileInput, optFns ...func(*bedrock.Options)) (*bedrock.GetInferenceProfileOutput, error) {
	return b.bedrockClient.GetInferenceProfile(ctx, params, optFns...)
}

func (b *bedrockClientImpl) DeleteInferenceProfile(ctx context.Context, params *bedrock.DeleteInferenceProfileInput, optFns ...func(*bedrock.Options)) (*bedrock.DeleteInferenceProfileOutput, error) {
	return b.bedrockClient.DeleteInferenceProfile(ctx, params, optFns...)
}

func (b *bedrockClientImpl) ListInferenceProfiles(ctx context.Context, params *bedrock.ListInferenceProfilesInput, optFns ...func(*bedrock.Options)) (*bedrock.ListInferenceProfilesOutput, error) {
	return b.bedrockClient.ListInferenceProfiles(ctx, params, optFns...)
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

// Inference Profile helper functions

// convertInferenceProfileSpecToCreateInput converts our InferenceProfileSpec to AWS CreateInferenceProfileInput
func convertInferenceProfileSpecToCreateInput(spec *InferenceProfileSpec) *bedrock.CreateInferenceProfileInput {
	input := &bedrock.CreateInferenceProfileInput{
		InferenceProfileName: &spec.Name,
	}

	if spec.Description != nil {
		input.Description = spec.Description
	}

	// Set model source - it's a union type with specific member
	input.ModelSource = &bedrocktypes.InferenceProfileModelSourceMemberCopyFrom{
		Value: spec.ModelSource.CopyFrom,
	}

	// Convert tags
	if len(spec.Tags) > 0 {
		var tags []bedrocktypes.Tag
		for key, value := range spec.Tags {
			tags = append(tags, bedrocktypes.Tag{
				Key:   &key,
				Value: &value,
			})
		}
		input.Tags = tags
	}

	return input
}

// convertAWSInferenceProfileToStatus converts AWS InferenceProfile to our status fields
func convertAWSInferenceProfileToStatus(profile *bedrocktypes.InferenceProfileSummary) *BedrockResourceStatus {
	status := &BedrockResourceStatus{
		Phase:               mapInferenceProfileStatusToPhase(profile.Status),
		InferenceProfileId:  profile.InferenceProfileId,
		InferenceProfileArn: profile.InferenceProfileArn,
		ResourceId:          profile.InferenceProfileId,
		ResourceArn:         profile.InferenceProfileArn,
	}

	if profile.Status != "" {
		status.InferenceProfileStatus = aws.String(string(profile.Status))
	}

	if profile.Type != "" {
		status.InferenceProfileType = aws.String(string(profile.Type))
	}

	return status
}

// mapInferenceProfileStatusToPhase maps AWS InferenceProfileStatus to our Phase
func mapInferenceProfileStatusToPhase(status bedrocktypes.InferenceProfileStatus) string {
	switch status {
	case bedrocktypes.InferenceProfileStatusActive:
		return PhasePrepared // Active means ready to use
	default:
		return PhaseCreating
	}
}

// Knowledge Base operations implementation
func (b *bedrockClientImpl) CreateKnowledgeBase(ctx context.Context, params *bedrockagent.CreateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateKnowledgeBaseOutput, error) {
	return b.agentClient.CreateKnowledgeBase(ctx, params, optFns...)
}

func (b *bedrockClientImpl) GetKnowledgeBase(ctx context.Context, params *bedrockagent.GetKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetKnowledgeBaseOutput, error) {
	return b.agentClient.GetKnowledgeBase(ctx, params, optFns...)
}

func (b *bedrockClientImpl) UpdateKnowledgeBase(ctx context.Context, params *bedrockagent.UpdateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateKnowledgeBaseOutput, error) {
	return b.agentClient.UpdateKnowledgeBase(ctx, params, optFns...)
}

func (b *bedrockClientImpl) DeleteKnowledgeBase(ctx context.Context, params *bedrockagent.DeleteKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteKnowledgeBaseOutput, error) {
	return b.agentClient.DeleteKnowledgeBase(ctx, params, optFns...)
}

func (b *bedrockClientImpl) ListKnowledgeBases(ctx context.Context, params *bedrockagent.ListKnowledgeBasesInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.ListKnowledgeBasesOutput, error) {
	return b.agentClient.ListKnowledgeBases(ctx, params, optFns...)
}

// Data Source operations implementation
func (b *bedrockClientImpl) CreateDataSource(ctx context.Context, params *bedrockagent.CreateDataSourceInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateDataSourceOutput, error) {
	return b.agentClient.CreateDataSource(ctx, params, optFns...)
}

func (b *bedrockClientImpl) GetDataSource(ctx context.Context, params *bedrockagent.GetDataSourceInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetDataSourceOutput, error) {
	return b.agentClient.GetDataSource(ctx, params, optFns...)
}

func (b *bedrockClientImpl) UpdateDataSource(ctx context.Context, params *bedrockagent.UpdateDataSourceInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateDataSourceOutput, error) {
	return b.agentClient.UpdateDataSource(ctx, params, optFns...)
}

func (b *bedrockClientImpl) DeleteDataSource(ctx context.Context, params *bedrockagent.DeleteDataSourceInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteDataSourceOutput, error) {
	return b.agentClient.DeleteDataSource(ctx, params, optFns...)
}

func (b *bedrockClientImpl) ListDataSources(ctx context.Context, params *bedrockagent.ListDataSourcesInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.ListDataSourcesOutput, error) {
	return b.agentClient.ListDataSources(ctx, params, optFns...)
}

func (b *bedrockClientImpl) StartIngestionJob(ctx context.Context, params *bedrockagent.StartIngestionJobInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.StartIngestionJobOutput, error) {
	return b.agentClient.StartIngestionJob(ctx, params, optFns...)
}

func (b *bedrockClientImpl) ListIngestionJobs(ctx context.Context, params *bedrockagent.ListIngestionJobsInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.ListIngestionJobsOutput, error) {
	return b.agentClient.ListIngestionJobs(ctx, params, optFns...)
}

// Helper functions for Knowledge Base operations

// ConvertKnowledgeBaseSpecToCreateInput converts our KnowledgeBaseSpec to AWS CreateKnowledgeBaseInput
func ConvertKnowledgeBaseSpecToCreateInput(spec *KnowledgeBaseSpec) *bedrockagent.CreateKnowledgeBaseInput {
	input := &bedrockagent.CreateKnowledgeBaseInput{
		Name:    aws.String(spec.Name),
		RoleArn: aws.String(spec.RoleArn),
		KnowledgeBaseConfiguration: &types.KnowledgeBaseConfiguration{
			Type: types.KnowledgeBaseTypeVector,
			VectorKnowledgeBaseConfiguration: &types.VectorKnowledgeBaseConfiguration{
				EmbeddingModelArn: aws.String(spec.EmbeddingModelArn),
			},
		},
	}

	// Set description if provided
	if spec.Description != nil {
		input.Description = spec.Description
	}

	// Set storage configuration based on vector store type
	switch spec.VectorStoreType {
	case "OPENSEARCH_SERVERLESS":
		if spec.OpenSearchServerlessConfiguration != nil {
			input.StorageConfiguration = &types.StorageConfiguration{
				Type: types.KnowledgeBaseStorageTypeOpensearchServerless,
				OpensearchServerlessConfiguration: &types.OpenSearchServerlessConfiguration{
					CollectionArn:   aws.String(spec.OpenSearchServerlessConfiguration.CollectionArn),
					VectorIndexName: aws.String(spec.OpenSearchServerlessConfiguration.VectorIndexName),
					FieldMapping: &types.OpenSearchServerlessFieldMapping{
						VectorField:   aws.String(spec.OpenSearchServerlessConfiguration.VectorField),
						TextField:     aws.String(spec.OpenSearchServerlessConfiguration.TextField),
						MetadataField: aws.String(spec.OpenSearchServerlessConfiguration.MetadataField),
					},
				},
			}
		}
		// Future: Add support for other vector store types (Pinecone, Redis, etc.)
	}

	// Set tags if provided
	if spec.Tags != nil {
		input.Tags = make(map[string]string)
		for k, v := range spec.Tags {
			input.Tags[k] = v
		}
	}

	return input
}

// ConvertAWSKnowledgeBaseToStatus converts AWS Knowledge Base to our status fields
func ConvertAWSKnowledgeBaseToStatus(kb *types.KnowledgeBase) *BedrockResourceStatus {
	status := &BedrockResourceStatus{
		Phase:            MapKnowledgeBaseStatusToPhase(kb.Status),
		KnowledgeBaseId:  kb.KnowledgeBaseId,
		KnowledgeBaseArn: kb.KnowledgeBaseArn,
		ResourceId:       kb.KnowledgeBaseId,
		ResourceArn:      kb.KnowledgeBaseArn,
	}

	if kb.Status != "" {
		status.KnowledgeBaseStatus = aws.String(string(kb.Status))
	}

	// Add failure reasons if any
	if len(kb.FailureReasons) > 0 {
		status.FailureReasons = make([]string, len(kb.FailureReasons))
		copy(status.FailureReasons, kb.FailureReasons)
	}

	return status
}

// MapKnowledgeBaseStatusToPhase maps AWS KnowledgeBaseStatus to our Phase
func MapKnowledgeBaseStatusToPhase(status types.KnowledgeBaseStatus) string {
	switch status {
	case types.KnowledgeBaseStatusActive:
		return PhasePrepared // Active means ready to use
	case types.KnowledgeBaseStatusCreating:
		return PhaseCreating
	case types.KnowledgeBaseStatusUpdating:
		return PhaseUpdating
	case types.KnowledgeBaseStatusDeleting:
		return PhaseDeleting
	case types.KnowledgeBaseStatusFailed:
		return PhaseFailed
	default:
		return PhaseCreating
	}
}

// OpenSearch operations implementation

// CheckOpenSearchIndexExists checks if an index exists in OpenSearch Serverless
func (b *bedrockClientImpl) CheckOpenSearchIndexExists(ctx context.Context, collectionId, indexName string) (bool, error) {
	// Use the AWS API to get index information
	_, err := b.opensearchServerlessClient.GetIndex(ctx, &opensearchserverless.GetIndexInput{
		Id:        aws.String(collectionId),
		IndexName: aws.String(indexName),
	})

	if err != nil {
		// If index doesn't exist, GetIndex returns an error
		// Check if it's a "not found" type error
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check index existence: %w", err)
	}

	return true, nil
}

// CreateOpenSearchIndex creates a vector index in OpenSearch Serverless using AWS API
func (b *bedrockClientImpl) CreateOpenSearchIndex(ctx context.Context, collectionId, indexName string, vectorDimension int) error {
	// Check if index already exists
	exists, err := b.CheckOpenSearchIndexExists(ctx, collectionId, indexName)
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}
	if exists {
		return nil // Index already exists, nothing to do
	}

	// Create index schema optimized for Bedrock Knowledge Base
	indexSchema := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"vector": map[string]interface{}{
					"type":      "knn_vector",
					"dimension": vectorDimension,
					"method": map[string]interface{}{
						"name":   "hnsw",
						"engine": "faiss",
						"parameters": map[string]interface{}{
							"ef_construction": 512,
							"m":               16,
						},
					},
				},
				"text": map[string]interface{}{
					"type": "text",
				},
				"metadata": map[string]interface{}{
					"type":    "object",
					"dynamic": true,
				},
			},
		},
		"settings": map[string]interface{}{
			"index": map[string]interface{}{
				"knn": true,
			},
		},
	}

	// Convert the schema to the document interface type expected by the SDK
	schemaDoc := document.NewLazyDocument(indexSchema)

	// Create the index using AWS OpenSearch Serverless CreateIndex API
	_, err = b.opensearchServerlessClient.CreateIndex(ctx, &opensearchserverless.CreateIndexInput{
		Id:          aws.String(collectionId),
		IndexName:   aws.String(indexName),
		IndexSchema: schemaDoc,
	})

	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}
