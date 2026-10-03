package tests

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
)

// MockBedrockClient is a mock implementation of controller.BedrockClient for testing
type MockBedrockClient struct {
	// Mock state
	agents            map[string]*types.Agent
	inferenceProfiles map[string]bedrocktypes.InferenceProfileSummary
	knowledgeBases    map[string]*types.KnowledgeBase
	dataSources       map[string]*types.DataSource

	// Control behavior - Agents
	createAgentError  error
	getAgentError     error
	updateAgentError  error
	deleteAgentError  error
	prepareAgentError error

	// Control behavior - Inference Profiles
	createInferenceProfileError error
	getInferenceProfileError    error
	deleteInferenceProfileError error
	listInferenceProfilesError  error

	// Control behavior - Knowledge Bases
	createKnowledgeBaseError error
	getKnowledgeBaseError    error
	updateKnowledgeBaseError error
	deleteKnowledgeBaseError error
	listKnowledgeBasesError  error

	// Control behavior - Data Sources
	createDataSourceError error
	getDataSourceError    error
	updateDataSourceError error
	deleteDataSourceError error
	listDataSourcesError  error

	// Control behavior - Ingestion Jobs
	startIngestionJobError error
	listIngestionJobsError error

	// Mock ingestion job state
	ingestionJobs map[string][]types.IngestionJobSummary // map[dataSourceId] = jobs

	// Control behavior - OpenSearch
	createOpenSearchIndexError      error
	checkOpenSearchIndexExistsError error

	// Mock OpenSearch state
	openSearchIndices map[string]bool // map[endpoint+index] = exists

	// Call tracking - Agents
	CreateAgentCalled  bool
	GetAgentCalled     bool
	UpdateAgentCalled  bool
	DeleteAgentCalled  bool
	PrepareAgentCalled bool

	// Call tracking - Inference Profiles
	CreateInferenceProfileCalled bool
	GetInferenceProfileCalled    bool
	DeleteInferenceProfileCalled bool
	ListInferenceProfilesCalled  bool

	// Call tracking - Knowledge Bases
	CreateKnowledgeBaseCalled bool
	GetKnowledgeBaseCalled    bool
	UpdateKnowledgeBaseCalled bool
	DeleteKnowledgeBaseCalled bool
	ListKnowledgeBasesCalled  bool

	// Call tracking - Data Sources
	CreateDataSourceCalled bool
	GetDataSourceCalled    bool
	UpdateDataSourceCalled bool
	DeleteDataSourceCalled bool
	ListDataSourcesCalled  bool

	// Call tracking - Ingestion Jobs
	StartIngestionJobCalled bool
	ListIngestionJobsCalled bool

	// Input tracking - Agents
	CreateAgentInput  *bedrockagent.CreateAgentInput
	GetAgentInput     *bedrockagent.GetAgentInput
	UpdateAgentInput  *bedrockagent.UpdateAgentInput
	DeleteAgentInput  *bedrockagent.DeleteAgentInput
	PrepareAgentInput *bedrockagent.PrepareAgentInput

	// Input tracking - Inference Profiles
	CreateInferenceProfileInput *bedrock.CreateInferenceProfileInput
	GetInferenceProfileInput    *bedrock.GetInferenceProfileInput
	DeleteInferenceProfileInput *bedrock.DeleteInferenceProfileInput
	ListInferenceProfilesInput  *bedrock.ListInferenceProfilesInput

	// Input tracking - Knowledge Bases
	CreateKnowledgeBaseInput *bedrockagent.CreateKnowledgeBaseInput
	GetKnowledgeBaseInput    *bedrockagent.GetKnowledgeBaseInput
	UpdateKnowledgeBaseInput *bedrockagent.UpdateKnowledgeBaseInput
	DeleteKnowledgeBaseInput *bedrockagent.DeleteKnowledgeBaseInput
	ListKnowledgeBasesInput  *bedrockagent.ListKnowledgeBasesInput

	// Input tracking - Data Sources
	CreateDataSourceInput *bedrockagent.CreateDataSourceInput
	GetDataSourceInput    *bedrockagent.GetDataSourceInput
	UpdateDataSourceInput *bedrockagent.UpdateDataSourceInput
	DeleteDataSourceInput *bedrockagent.DeleteDataSourceInput
	ListDataSourcesInput  *bedrockagent.ListDataSourcesInput

	// Input tracking - Ingestion Jobs
	StartIngestionJobInput *bedrockagent.StartIngestionJobInput
	ListIngestionJobsInput *bedrockagent.ListIngestionJobsInput
}

// NewMockBedrockClient creates a new mock Bedrock client
func NewMockBedrockClient() *MockBedrockClient {
	return &MockBedrockClient{
		agents:            make(map[string]*types.Agent),
		inferenceProfiles: make(map[string]bedrocktypes.InferenceProfileSummary),
		knowledgeBases:    make(map[string]*types.KnowledgeBase),
		dataSources:       make(map[string]*types.DataSource),
		ingestionJobs:     make(map[string][]types.IngestionJobSummary),
		openSearchIndices: make(map[string]bool),
	}
}

// CreateAgent implements BedrockClient.CreateAgent
func (m *MockBedrockClient) CreateAgent(ctx context.Context, params *bedrockagent.CreateAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateAgentOutput, error) {
	if m.createAgentError != nil {
		return nil, m.createAgentError
	}

	agentId := fmt.Sprintf("agent-%d", len(m.agents)+1)
	agentArn := fmt.Sprintf("arn:aws:bedrock:us-east-1:123456789012:agent/%s", agentId)

	agent := &types.Agent{
		AgentId:                  &agentId,
		AgentArn:                 &agentArn,
		AgentName:                params.AgentName,
		AgentStatus:              types.AgentStatusNotPrepared,
		AgentVersion:             &[]string{"DRAFT"}[0],
		AgentResourceRoleArn:     params.AgentResourceRoleArn,
		Description:              params.Description,
		FoundationModel:          params.FoundationModel,
		Instruction:              params.Instruction,
		CustomerEncryptionKeyArn: params.CustomerEncryptionKeyArn,
		IdleSessionTTLInSeconds:  params.IdleSessionTTLInSeconds,
		GuardrailConfiguration:   params.GuardrailConfiguration,
		MemoryConfiguration:      params.MemoryConfiguration,
	}

	m.agents[agentId] = agent

	return &bedrockagent.CreateAgentOutput{
		Agent: agent,
	}, nil
}

// GetAgent implements BedrockClient.GetAgent
func (m *MockBedrockClient) GetAgent(ctx context.Context, params *bedrockagent.GetAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetAgentOutput, error) {
	if m.getAgentError != nil {
		return nil, m.getAgentError
	}

	agent, exists := m.agents[*params.AgentId]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", *params.AgentId)
	}

	return &bedrockagent.GetAgentOutput{
		Agent: agent,
	}, nil
}

// UpdateAgent implements BedrockClient.UpdateAgent
func (m *MockBedrockClient) UpdateAgent(ctx context.Context, params *bedrockagent.UpdateAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateAgentOutput, error) {
	if m.updateAgentError != nil {
		return nil, m.updateAgentError
	}

	agent, exists := m.agents[*params.AgentId]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", *params.AgentId)
	}

	// Update agent fields
	if params.AgentName != nil {
		agent.AgentName = params.AgentName
	}
	if params.Description != nil {
		agent.Description = params.Description
	}
	if params.FoundationModel != nil {
		agent.FoundationModel = params.FoundationModel
	}
	if params.Instruction != nil {
		agent.Instruction = params.Instruction
	}
	if params.AgentResourceRoleArn != nil {
		agent.AgentResourceRoleArn = params.AgentResourceRoleArn
	}
	if params.CustomerEncryptionKeyArn != nil {
		agent.CustomerEncryptionKeyArn = params.CustomerEncryptionKeyArn
	}
	if params.IdleSessionTTLInSeconds != nil {
		agent.IdleSessionTTLInSeconds = params.IdleSessionTTLInSeconds
	}
	if params.GuardrailConfiguration != nil {
		agent.GuardrailConfiguration = params.GuardrailConfiguration
	}
	if params.MemoryConfiguration != nil {
		agent.MemoryConfiguration = params.MemoryConfiguration
	}

	agent.AgentStatus = types.AgentStatusNotPrepared

	return &bedrockagent.UpdateAgentOutput{
		Agent: agent,
	}, nil
}

// DeleteAgent implements BedrockClient.DeleteAgent
func (m *MockBedrockClient) DeleteAgent(ctx context.Context, params *bedrockagent.DeleteAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteAgentOutput, error) {
	m.DeleteAgentCalled = true
	m.DeleteAgentInput = params

	if m.deleteAgentError != nil {
		return nil, m.deleteAgentError
	}

	agent, exists := m.agents[*params.AgentId]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", *params.AgentId)
	}

	agent.AgentStatus = types.AgentStatusDeleting
	delete(m.agents, *params.AgentId)

	return &bedrockagent.DeleteAgentOutput{
		AgentId:     agent.AgentId,
		AgentStatus: agent.AgentStatus,
	}, nil
}

// PrepareAgent implements BedrockClient.PrepareAgent
func (m *MockBedrockClient) PrepareAgent(ctx context.Context, params *bedrockagent.PrepareAgentInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.PrepareAgentOutput, error) {
	if m.prepareAgentError != nil {
		return nil, m.prepareAgentError
	}

	agent, exists := m.agents[*params.AgentId]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", *params.AgentId)
	}

	agent.AgentStatus = types.AgentStatusPrepared

	return &bedrockagent.PrepareAgentOutput{
		AgentId:      agent.AgentId,
		AgentStatus:  agent.AgentStatus,
		AgentVersion: agent.AgentVersion,
	}, nil
}

// Test helper methods

// SetCreateAgentError sets an error to be returned by CreateAgent
func (m *MockBedrockClient) SetCreateAgentError(err error) {
	m.createAgentError = err
}

// SetGetAgentError sets an error to be returned by GetAgent
func (m *MockBedrockClient) SetGetAgentError(err error) {
	m.getAgentError = err
}

// SetUpdateAgentError sets an error to be returned by UpdateAgent
func (m *MockBedrockClient) SetUpdateAgentError(err error) {
	m.updateAgentError = err
}

// SetDeleteAgentError sets an error to be returned by DeleteAgent
func (m *MockBedrockClient) SetDeleteAgentError(err error) {
	m.deleteAgentError = err
}

// SetPrepareAgentError sets an error to be returned by PrepareAgent
func (m *MockBedrockClient) SetPrepareAgentError(err error) {
	m.prepareAgentError = err
}

// AddAgent adds an agent to the mock state for testing
func (m *MockBedrockClient) AddAgent(agent *types.Agent) {
	if m.agents == nil {
		m.agents = make(map[string]*types.Agent)
	}
	m.agents[*agent.AgentId] = agent
}

// GetAgents returns all agents in the mock
func (m *MockBedrockClient) GetAgents() map[string]*types.Agent {
	return m.agents
}

// Inference Profile operations implementation

// CreateInferenceProfile implements BedrockClient.CreateInferenceProfile
func (m *MockBedrockClient) CreateInferenceProfile(ctx context.Context, params *bedrock.CreateInferenceProfileInput, optFns ...func(*bedrock.Options)) (*bedrock.CreateInferenceProfileOutput, error) {
	m.CreateInferenceProfileCalled = true
	m.CreateInferenceProfileInput = params

	if m.createInferenceProfileError != nil {
		return nil, m.createInferenceProfileError
	}

	// Generate a unique mock profile ID and ARN
	profileId := fmt.Sprintf("test-profile-%d", time.Now().UnixNano()%1000000)
	profileArn := fmt.Sprintf("arn:aws:bedrock:us-east-1:123456789012:inference-profile/%s", profileId)

	profile := bedrocktypes.InferenceProfileSummary{
		InferenceProfileId:   &profileId,
		InferenceProfileArn:  &profileArn,
		InferenceProfileName: params.InferenceProfileName,
		Status:               bedrocktypes.InferenceProfileStatusActive,
	}

	// Initialize map if it's nil
	if m.inferenceProfiles == nil {
		m.inferenceProfiles = make(map[string]bedrocktypes.InferenceProfileSummary)
	}
	m.inferenceProfiles[profileId] = profile

	return &bedrock.CreateInferenceProfileOutput{
		InferenceProfileArn: &profileArn,
		Status:              bedrocktypes.InferenceProfileStatusActive,
	}, nil
}

// GetInferenceProfile implements BedrockClient.GetInferenceProfile
func (m *MockBedrockClient) GetInferenceProfile(ctx context.Context, params *bedrock.GetInferenceProfileInput, optFns ...func(*bedrock.Options)) (*bedrock.GetInferenceProfileOutput, error) {
	m.GetInferenceProfileCalled = true
	m.GetInferenceProfileInput = params

	if m.getInferenceProfileError != nil {
		return nil, m.getInferenceProfileError
	}

	profile, exists := m.inferenceProfiles[*params.InferenceProfileIdentifier]
	if !exists {
		return nil, fmt.Errorf("inference profile not found: %s", *params.InferenceProfileIdentifier)
	}

	return &bedrock.GetInferenceProfileOutput{
		InferenceProfileId:   profile.InferenceProfileId,
		InferenceProfileArn:  profile.InferenceProfileArn,
		InferenceProfileName: profile.InferenceProfileName,
		Status:               profile.Status,
	}, nil
}

// DeleteInferenceProfile implements BedrockClient.DeleteInferenceProfile
func (m *MockBedrockClient) DeleteInferenceProfile(ctx context.Context, params *bedrock.DeleteInferenceProfileInput, optFns ...func(*bedrock.Options)) (*bedrock.DeleteInferenceProfileOutput, error) {
	m.DeleteInferenceProfileCalled = true
	m.DeleteInferenceProfileInput = params

	if m.deleteInferenceProfileError != nil {
		return nil, m.deleteInferenceProfileError
	}

	_, exists := m.inferenceProfiles[*params.InferenceProfileIdentifier]
	if !exists {
		return nil, fmt.Errorf("inference profile not found: %s", *params.InferenceProfileIdentifier)
	}

	delete(m.inferenceProfiles, *params.InferenceProfileIdentifier)

	return &bedrock.DeleteInferenceProfileOutput{}, nil
}

// ListInferenceProfiles implements BedrockClient.ListInferenceProfiles
func (m *MockBedrockClient) ListInferenceProfiles(ctx context.Context, params *bedrock.ListInferenceProfilesInput, optFns ...func(*bedrock.Options)) (*bedrock.ListInferenceProfilesOutput, error) {
	m.ListInferenceProfilesCalled = true
	m.ListInferenceProfilesInput = params

	if m.listInferenceProfilesError != nil {
		return nil, m.listInferenceProfilesError
	}

	var profiles []bedrocktypes.InferenceProfileSummary
	for _, profile := range m.inferenceProfiles {
		profiles = append(profiles, profile)
	}

	return &bedrock.ListInferenceProfilesOutput{
		InferenceProfileSummaries: profiles,
	}, nil
}

// Knowledge Base operations implementation

// CreateKnowledgeBase implements BedrockClient.CreateKnowledgeBase
func (m *MockBedrockClient) CreateKnowledgeBase(ctx context.Context, params *bedrockagent.CreateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateKnowledgeBaseOutput, error) {
	m.CreateKnowledgeBaseCalled = true
	m.CreateKnowledgeBaseInput = params

	if m.createKnowledgeBaseError != nil {
		return nil, m.createKnowledgeBaseError
	}

	// Generate a unique ID
	kbId := fmt.Sprintf("kb-%d", time.Now().UnixNano())
	kbArn := fmt.Sprintf("arn:aws:bedrock:us-east-1:123456789012:knowledge-base/%s", kbId)

	// Create the knowledge base object
	now := time.Now().UTC()
	kb := &types.KnowledgeBase{
		KnowledgeBaseId:            &kbId,
		KnowledgeBaseArn:           &kbArn,
		Name:                       params.Name,
		Description:                params.Description,
		RoleArn:                    params.RoleArn,
		KnowledgeBaseConfiguration: params.KnowledgeBaseConfiguration,
		StorageConfiguration:       params.StorageConfiguration,
		Status:                     types.KnowledgeBaseStatusActive,
		CreatedAt:                  &now,
		UpdatedAt:                  &now,
	}

	// Store in mock state
	if m.knowledgeBases == nil {
		m.knowledgeBases = make(map[string]*types.KnowledgeBase)
	}
	m.knowledgeBases[kbId] = kb

	return &bedrockagent.CreateKnowledgeBaseOutput{
		KnowledgeBase: kb,
	}, nil
}

// GetKnowledgeBase implements BedrockClient.GetKnowledgeBase
func (m *MockBedrockClient) GetKnowledgeBase(ctx context.Context, params *bedrockagent.GetKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetKnowledgeBaseOutput, error) {
	m.GetKnowledgeBaseCalled = true
	m.GetKnowledgeBaseInput = params

	if m.getKnowledgeBaseError != nil {
		return nil, m.getKnowledgeBaseError
	}

	kb, exists := m.knowledgeBases[*params.KnowledgeBaseId]
	if !exists {
		return nil, fmt.Errorf("ResourceNotFoundException: Knowledge base not found: %s", *params.KnowledgeBaseId)
	}

	return &bedrockagent.GetKnowledgeBaseOutput{
		KnowledgeBase: kb,
	}, nil
}

// UpdateKnowledgeBase implements BedrockClient.UpdateKnowledgeBase
func (m *MockBedrockClient) UpdateKnowledgeBase(ctx context.Context, params *bedrockagent.UpdateKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateKnowledgeBaseOutput, error) {
	m.UpdateKnowledgeBaseCalled = true
	m.UpdateKnowledgeBaseInput = params

	if m.updateKnowledgeBaseError != nil {
		return nil, m.updateKnowledgeBaseError
	}

	kb, exists := m.knowledgeBases[*params.KnowledgeBaseId]
	if !exists {
		return nil, fmt.Errorf("ResourceNotFoundException: Knowledge base not found: %s", *params.KnowledgeBaseId)
	}

	// Update fields
	if params.Name != nil {
		kb.Name = params.Name
	}
	if params.Description != nil {
		kb.Description = params.Description
	}
	if params.RoleArn != nil {
		kb.RoleArn = params.RoleArn
	}
	if params.KnowledgeBaseConfiguration != nil {
		kb.KnowledgeBaseConfiguration = params.KnowledgeBaseConfiguration
	}
	if params.StorageConfiguration != nil {
		kb.StorageConfiguration = params.StorageConfiguration
	}

	now := time.Now().UTC()
	kb.UpdatedAt = &now

	return &bedrockagent.UpdateKnowledgeBaseOutput{
		KnowledgeBase: kb,
	}, nil
}

// DeleteKnowledgeBase implements BedrockClient.DeleteKnowledgeBase
func (m *MockBedrockClient) DeleteKnowledgeBase(ctx context.Context, params *bedrockagent.DeleteKnowledgeBaseInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteKnowledgeBaseOutput, error) {
	m.DeleteKnowledgeBaseCalled = true
	m.DeleteKnowledgeBaseInput = params

	if m.deleteKnowledgeBaseError != nil {
		return nil, m.deleteKnowledgeBaseError
	}

	_, exists := m.knowledgeBases[*params.KnowledgeBaseId]
	if !exists {
		return nil, fmt.Errorf("ResourceNotFoundException: Knowledge base not found: %s", *params.KnowledgeBaseId)
	}

	delete(m.knowledgeBases, *params.KnowledgeBaseId)

	return &bedrockagent.DeleteKnowledgeBaseOutput{
		Status: types.KnowledgeBaseStatusDeleting,
	}, nil
}

// ListKnowledgeBases implements BedrockClient.ListKnowledgeBases
func (m *MockBedrockClient) ListKnowledgeBases(ctx context.Context, params *bedrockagent.ListKnowledgeBasesInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.ListKnowledgeBasesOutput, error) {
	m.ListKnowledgeBasesCalled = true
	m.ListKnowledgeBasesInput = params

	if m.listKnowledgeBasesError != nil {
		return nil, m.listKnowledgeBasesError
	}

	var summaries []types.KnowledgeBaseSummary
	for _, kb := range m.knowledgeBases {
		summary := types.KnowledgeBaseSummary{
			KnowledgeBaseId: kb.KnowledgeBaseId,
			Name:            kb.Name,
			Description:     kb.Description,
			Status:          kb.Status,
			UpdatedAt:       kb.UpdatedAt,
		}
		summaries = append(summaries, summary)
	}

	return &bedrockagent.ListKnowledgeBasesOutput{
		KnowledgeBaseSummaries: summaries,
	}, nil
}

// Data Source operations implementation (simplified for now)

// CreateDataSource implements BedrockClient.CreateDataSource
func (m *MockBedrockClient) CreateDataSource(ctx context.Context, params *bedrockagent.CreateDataSourceInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.CreateDataSourceOutput, error) {
	m.CreateDataSourceCalled = true
	m.CreateDataSourceInput = params

	if m.createDataSourceError != nil {
		return nil, m.createDataSourceError
	}

	// Generate a unique ID
	dsId := fmt.Sprintf("ds-%d", time.Now().UnixNano())

	// Create the data source object
	now := time.Now().UTC()
	ds := &types.DataSource{
		DataSourceId:            &dsId,
		KnowledgeBaseId:         params.KnowledgeBaseId,
		Name:                    params.Name,
		Description:             params.Description,
		DataSourceConfiguration: params.DataSourceConfiguration,
		Status:                  types.DataSourceStatusAvailable,
		CreatedAt:               &now,
		UpdatedAt:               &now,
	}

	// Store in mock state
	if m.dataSources == nil {
		m.dataSources = make(map[string]*types.DataSource)
	}
	m.dataSources[dsId] = ds

	return &bedrockagent.CreateDataSourceOutput{
		DataSource: ds,
	}, nil
}

// GetDataSource implements BedrockClient.GetDataSource
func (m *MockBedrockClient) GetDataSource(ctx context.Context, params *bedrockagent.GetDataSourceInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.GetDataSourceOutput, error) {
	m.GetDataSourceCalled = true
	m.GetDataSourceInput = params

	if m.getDataSourceError != nil {
		return nil, m.getDataSourceError
	}

	ds, exists := m.dataSources[*params.DataSourceId]
	if !exists {
		return nil, fmt.Errorf("ResourceNotFoundException: Data source not found: %s", *params.DataSourceId)
	}

	return &bedrockagent.GetDataSourceOutput{
		DataSource: ds,
	}, nil
}

// UpdateDataSource implements BedrockClient.UpdateDataSource
func (m *MockBedrockClient) UpdateDataSource(ctx context.Context, params *bedrockagent.UpdateDataSourceInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.UpdateDataSourceOutput, error) {
	m.UpdateDataSourceCalled = true
	m.UpdateDataSourceInput = params

	if m.updateDataSourceError != nil {
		return nil, m.updateDataSourceError
	}

	ds, exists := m.dataSources[*params.DataSourceId]
	if !exists {
		return nil, fmt.Errorf("ResourceNotFoundException: Data source not found: %s", *params.DataSourceId)
	}

	// Update fields
	if params.Name != nil {
		ds.Name = params.Name
	}
	if params.Description != nil {
		ds.Description = params.Description
	}
	if params.DataSourceConfiguration != nil {
		ds.DataSourceConfiguration = params.DataSourceConfiguration
	}

	now := time.Now().UTC()
	ds.UpdatedAt = &now

	return &bedrockagent.UpdateDataSourceOutput{
		DataSource: ds,
	}, nil
}

// DeleteDataSource implements BedrockClient.DeleteDataSource
func (m *MockBedrockClient) DeleteDataSource(ctx context.Context, params *bedrockagent.DeleteDataSourceInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.DeleteDataSourceOutput, error) {
	m.DeleteDataSourceCalled = true
	m.DeleteDataSourceInput = params

	if m.deleteDataSourceError != nil {
		return nil, m.deleteDataSourceError
	}

	_, exists := m.dataSources[*params.DataSourceId]
	if !exists {
		return nil, fmt.Errorf("ResourceNotFoundException: Data source not found: %s", *params.DataSourceId)
	}

	delete(m.dataSources, *params.DataSourceId)

	return &bedrockagent.DeleteDataSourceOutput{
		Status: types.DataSourceStatusDeleting,
	}, nil
}

// ListDataSources implements BedrockClient.ListDataSources
func (m *MockBedrockClient) ListDataSources(ctx context.Context, params *bedrockagent.ListDataSourcesInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.ListDataSourcesOutput, error) {
	m.ListDataSourcesCalled = true
	m.ListDataSourcesInput = params

	if m.listDataSourcesError != nil {
		return nil, m.listDataSourcesError
	}

	var summaries []types.DataSourceSummary
	for _, ds := range m.dataSources {
		if *ds.KnowledgeBaseId == *params.KnowledgeBaseId {
			summary := types.DataSourceSummary{
				DataSourceId:    ds.DataSourceId,
				Name:            ds.Name,
				Description:     ds.Description,
				Status:          ds.Status,
				KnowledgeBaseId: ds.KnowledgeBaseId,
				UpdatedAt:       ds.UpdatedAt,
			}
			summaries = append(summaries, summary)
		}
	}

	return &bedrockagent.ListDataSourcesOutput{
		DataSourceSummaries: summaries,
	}, nil
}

// StartIngestionJob implements BedrockClient.StartIngestionJob
func (m *MockBedrockClient) StartIngestionJob(ctx context.Context, params *bedrockagent.StartIngestionJobInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.StartIngestionJobOutput, error) {
	m.StartIngestionJobCalled = true
	m.StartIngestionJobInput = params

	if m.startIngestionJobError != nil {
		return nil, m.startIngestionJobError
	}

	jobId := fmt.Sprintf("job-%d", time.Now().UnixNano())
	now := time.Now()
	summary := types.IngestionJobSummary{
		IngestionJobId:  aws.String(jobId),
		KnowledgeBaseId: params.KnowledgeBaseId,
		DataSourceId:    params.DataSourceId,
		Status:          types.IngestionJobStatusStarting,
		StartedAt:       &now,
		UpdatedAt:       &now,
	}
	if m.ingestionJobs == nil {
		m.ingestionJobs = make(map[string][]types.IngestionJobSummary)
	}
	m.ingestionJobs[*params.DataSourceId] = append(m.ingestionJobs[*params.DataSourceId], summary)

	return &bedrockagent.StartIngestionJobOutput{
		IngestionJob: &types.IngestionJob{
			IngestionJobId:  summary.IngestionJobId,
			KnowledgeBaseId: summary.KnowledgeBaseId,
			DataSourceId:    summary.DataSourceId,
			Status:          summary.Status,
			StartedAt:       summary.StartedAt,
			UpdatedAt:       summary.UpdatedAt,
		},
	}, nil
}

// ListIngestionJobs implements BedrockClient.ListIngestionJobs
func (m *MockBedrockClient) ListIngestionJobs(ctx context.Context, params *bedrockagent.ListIngestionJobsInput, optFns ...func(*bedrockagent.Options)) (*bedrockagent.ListIngestionJobsOutput, error) {
	m.ListIngestionJobsCalled = true
	m.ListIngestionJobsInput = params

	if m.listIngestionJobsError != nil {
		return nil, m.listIngestionJobsError
	}

	return &bedrockagent.ListIngestionJobsOutput{
		IngestionJobSummaries: m.ingestionJobs[*params.DataSourceId],
	}, nil
}

// SetStartIngestionJobError sets an error to be returned by StartIngestionJob
func (m *MockBedrockClient) SetStartIngestionJobError(err error) {
	m.startIngestionJobError = err
}

// SetListIngestionJobsError sets an error to be returned by ListIngestionJobs
func (m *MockBedrockClient) SetListIngestionJobsError(err error) {
	m.listIngestionJobsError = err
}

// Helper methods for setting up inference profile errors

// SetCreateInferenceProfileError sets an error to be returned by CreateInferenceProfile
func (m *MockBedrockClient) SetCreateInferenceProfileError(err error) {
	m.createInferenceProfileError = err
}

// SetGetInferenceProfileError sets an error to be returned by GetInferenceProfile
func (m *MockBedrockClient) SetGetInferenceProfileError(err error) {
	m.getInferenceProfileError = err
}

// SetDeleteInferenceProfileError sets an error to be returned by DeleteInferenceProfile
func (m *MockBedrockClient) SetDeleteInferenceProfileError(err error) {
	m.deleteInferenceProfileError = err
}

// SetListInferenceProfilesError sets an error to be returned by ListInferenceProfiles
func (m *MockBedrockClient) SetListInferenceProfilesError(err error) {
	m.listInferenceProfilesError = err
}

// AddInferenceProfile adds an inference profile to the mock state for testing
func (m *MockBedrockClient) AddInferenceProfile(profile *bedrocktypes.InferenceProfileSummary) {
	if m.inferenceProfiles == nil {
		m.inferenceProfiles = make(map[string]bedrocktypes.InferenceProfileSummary)
	}
	m.inferenceProfiles[*profile.InferenceProfileId] = *profile
}

// GetInferenceProfiles returns all inference profiles in the mock
func (m *MockBedrockClient) GetInferenceProfiles() map[string]bedrocktypes.InferenceProfileSummary {
	return m.inferenceProfiles
}

// AddKnowledgeBase adds a knowledge base to the mock state for testing
func (m *MockBedrockClient) AddKnowledgeBase(kb *types.KnowledgeBase) {
	if m.knowledgeBases == nil {
		m.knowledgeBases = make(map[string]*types.KnowledgeBase)
	}
	m.knowledgeBases[*kb.KnowledgeBaseId] = kb
}

// GetKnowledgeBases returns all knowledge bases in the mock
func (m *MockBedrockClient) GetKnowledgeBases() map[string]*types.KnowledgeBase {
	return m.knowledgeBases
}

// Helper methods for setting up Knowledge Base errors

// SetCreateKnowledgeBaseError sets an error to be returned by CreateKnowledgeBase
func (m *MockBedrockClient) SetCreateKnowledgeBaseError(err error) {
	m.createKnowledgeBaseError = err
}

// SetGetKnowledgeBaseError sets an error to be returned by GetKnowledgeBase
func (m *MockBedrockClient) SetGetKnowledgeBaseError(err error) {
	m.getKnowledgeBaseError = err
}

// SetUpdateKnowledgeBaseError sets an error to be returned by UpdateKnowledgeBase
func (m *MockBedrockClient) SetUpdateKnowledgeBaseError(err error) {
	m.updateKnowledgeBaseError = err
}

// SetDeleteKnowledgeBaseError sets an error to be returned by DeleteKnowledgeBase
func (m *MockBedrockClient) SetDeleteKnowledgeBaseError(err error) {
	m.deleteKnowledgeBaseError = err
}

// SetListKnowledgeBasesError sets an error to be returned by ListKnowledgeBases
func (m *MockBedrockClient) SetListKnowledgeBasesError(err error) {
	m.listKnowledgeBasesError = err
}

// Reset clears all mock state
func (m *MockBedrockClient) Reset() {
	m.agents = make(map[string]*types.Agent)
	m.inferenceProfiles = make(map[string]bedrocktypes.InferenceProfileSummary)
	m.knowledgeBases = make(map[string]*types.KnowledgeBase)
	m.dataSources = make(map[string]*types.DataSource)
	m.ingestionJobs = make(map[string][]types.IngestionJobSummary)

	// Clear agent errors
	m.createAgentError = nil
	m.getAgentError = nil
	m.updateAgentError = nil
	m.deleteAgentError = nil
	m.prepareAgentError = nil

	// Clear inference profile errors
	m.createInferenceProfileError = nil
	m.getInferenceProfileError = nil
	m.deleteInferenceProfileError = nil
	m.listInferenceProfilesError = nil

	// Clear knowledge base errors
	m.createKnowledgeBaseError = nil
	m.getKnowledgeBaseError = nil
	m.updateKnowledgeBaseError = nil
	m.deleteKnowledgeBaseError = nil
	m.listKnowledgeBasesError = nil

	// Clear data source errors
	m.createDataSourceError = nil
	m.getDataSourceError = nil
	m.updateDataSourceError = nil
	m.deleteDataSourceError = nil
	m.listDataSourcesError = nil

	// Clear ingestion job errors
	m.startIngestionJobError = nil
	m.listIngestionJobsError = nil

	// Clear OpenSearch errors
	m.createOpenSearchIndexError = nil
	m.checkOpenSearchIndexExistsError = nil

	// Clear OpenSearch indices
	m.openSearchIndices = make(map[string]bool)
}

// OpenSearch methods implementation

// CreateOpenSearchIndex implements BedrockClient.CreateOpenSearchIndex
func (m *MockBedrockClient) CreateOpenSearchIndex(ctx context.Context, collectionId, indexName string, vectorDimension int) error {
	if m.createOpenSearchIndexError != nil {
		return m.createOpenSearchIndexError
	}

	key := collectionId + "/" + indexName
	m.openSearchIndices[key] = true
	return nil
}

// CheckOpenSearchIndexExists implements BedrockClient.CheckOpenSearchIndexExists
func (m *MockBedrockClient) CheckOpenSearchIndexExists(ctx context.Context, collectionId, indexName string) (bool, error) {
	if m.checkOpenSearchIndexExistsError != nil {
		return false, m.checkOpenSearchIndexExistsError
	}

	key := collectionId + "/" + indexName
	exists, found := m.openSearchIndices[key]
	return found && exists, nil
}

// Helper methods for OpenSearch testing

// SetCreateOpenSearchIndexError sets an error to be returned by CreateOpenSearchIndex
func (m *MockBedrockClient) SetCreateOpenSearchIndexError(err error) {
	m.createOpenSearchIndexError = err
}

// SetCheckOpenSearchIndexExistsError sets an error to be returned by CheckOpenSearchIndexExists
func (m *MockBedrockClient) SetCheckOpenSearchIndexExistsError(err error) {
	m.checkOpenSearchIndexExistsError = err
}

// AddOpenSearchIndex manually adds an index to the mock (for test setup)
func (m *MockBedrockClient) AddOpenSearchIndex(collectionId, indexName string) {
	key := collectionId + "/" + indexName
	m.openSearchIndices[key] = true
}
