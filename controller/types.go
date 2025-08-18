package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BedrockResource is the Schema for the bedrockresources API
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BedrockResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BedrockResourceSpec   `json:"spec,omitempty"`
	Status BedrockResourceStatus `json:"status,omitempty"`
}

// BedrockResourceList contains a list of BedrockResource
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BedrockResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BedrockResource `json:"items"`
}

// BedrockResourceSpec defines the desired state of BedrockResource
type BedrockResourceSpec struct {
	// Type specifies the type of Bedrock resource to manage
	// +kubebuilder:validation:Enum=agent;inferenceProfile;knowledgeBase
	Type string `json:"type"`

	// Agent configuration when Type is "agent"
	// +optional
	Agent *AgentSpec `json:"agent,omitempty"`

	// InferenceProfile configuration when Type is "inferenceProfile"
	// +optional
	InferenceProfile *InferenceProfileSpec `json:"inferenceProfile,omitempty"`

	// KnowledgeBase configuration when Type is "knowledgeBase"
	// +optional
	KnowledgeBase *KnowledgeBaseSpec `json:"knowledgeBase,omitempty"`

	// Region specifies the AWS region for the Bedrock resource
	// +kubebuilder:default="us-east-1"
	// +optional
	Region string `json:"region,omitempty"`
}

// AgentSpec defines the configuration for a Bedrock Agent
type AgentSpec struct {
	// Name for the agent
	Name string `json:"name"`

	// Description of the agent
	// +optional
	Description *string `json:"description,omitempty"`

	// FoundationModel ID for the agent
	// +optional
	FoundationModel *string `json:"foundationModel,omitempty"`

	// Instruction tells the agent what it should do
	// +optional
	Instruction *string `json:"instruction,omitempty"`

	// AgentResourceRoleArn is the IAM role ARN with permissions to invoke API operations
	// +optional
	AgentResourceRoleArn *string `json:"agentResourceRoleArn,omitempty"`

	// CustomerEncryptionKeyArn is the KMS key ARN for encryption
	// +optional
	CustomerEncryptionKeyArn *string `json:"customerEncryptionKeyArn,omitempty"`

	// IdleSessionTTLInSeconds is the session timeout in seconds
	// +kubebuilder:validation:Minimum=60
	// +kubebuilder:validation:Maximum=3600
	// +optional
	IdleSessionTTLInSeconds *int32 `json:"idleSessionTTLInSeconds,omitempty"`

	// AutoPrepare automatically prepares the agent after creation/updates
	// +kubebuilder:default=false
	// +optional
	AutoPrepare bool `json:"autoPrepare,omitempty"`

	// Tags are AWS tags to apply to the agent
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// GuardrailConfiguration for the agent
	// +optional
	GuardrailConfiguration *GuardrailConfiguration `json:"guardrailConfiguration,omitempty"`

	// MemoryConfiguration for the agent
	// +optional
	MemoryConfiguration *MemoryConfiguration `json:"memoryConfiguration,omitempty"`
}

// GuardrailConfiguration represents guardrail settings for an agent
type GuardrailConfiguration struct {
	// GuardrailId is the ID of the guardrail
	// +optional
	GuardrailId *string `json:"guardrailId,omitempty"`

	// GuardrailVersion is the version of the guardrail
	// +optional
	GuardrailVersion *string `json:"guardrailVersion,omitempty"`
}

// MemoryConfiguration represents memory settings for an agent
type MemoryConfiguration struct {
	// EnabledMemoryTypes specifies which memory types are enabled
	// +optional
	EnabledMemoryTypes []MemoryType `json:"enabledMemoryTypes,omitempty"`

	// StorageDays specifies how many days to store memory
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=999
	// +optional
	StorageDays *int32 `json:"storageDays,omitempty"`
}

// MemoryType represents the type of memory enabled for an agent
// +kubebuilder:validation:Enum=SESSION_SUMMARY
type MemoryType string

const (
	MemoryTypeSessionSummary MemoryType = "SESSION_SUMMARY"
)

// InferenceProfileSpec defines the configuration for a Bedrock Inference Profile
type InferenceProfileSpec struct {
	// Name for the inference profile
	Name string `json:"name"`

	// Description of the inference profile
	// +optional
	Description *string `json:"description,omitempty"`

	// ModelSource specifies the foundation model or system-defined inference profile
	// that this inference profile will track metrics and costs for
	ModelSource InferenceProfileModelSource `json:"modelSource"`

	// Tags to apply to the inference profile
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// InferenceProfileModelSource contains information about the model or system-defined
// inference profile that is the source for an inference profile
type InferenceProfileModelSource struct {
	// CopyFrom is the ARN of the model or system-defined inference profile
	// that is the source for the inference profile
	CopyFrom string `json:"copyFrom"`
}

// KnowledgeBaseSpec defines the configuration for a Bedrock Knowledge Base
type KnowledgeBaseSpec struct {
	// Name for the knowledge base
	Name string `json:"name"`

	// Description of the knowledge base
	// +optional
	Description *string `json:"description,omitempty"`

	// RoleArn is the IAM role ARN that the Knowledge Base uses to access other AWS services
	RoleArn string `json:"roleArn"`

	// EmbeddingModelArn is the ARN of the model used for generating embeddings
	EmbeddingModelArn string `json:"embeddingModelArn"`

	// VectorStoreType defines the type of vector database to use
	// +kubebuilder:validation:Enum=OPENSEARCH_SERVERLESS;PINECONE;REDIS_ENTERPRISE_CLOUD
	VectorStoreType string `json:"vectorStoreType"`

	// OpenSearchServerlessConfiguration for OpenSearch Serverless (most common)
	// +optional
	OpenSearchServerlessConfiguration *OpenSearchServerlessConfig `json:"opensearchServerlessConfiguration,omitempty"`

	// DataSources defines the data sources to be associated with this knowledge base
	// +optional
	DataSources []DataSourceSpec `json:"dataSources,omitempty"`

	// Tags for resource organization and cost tracking
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// OpenSearchServerlessConfig defines OpenSearch Serverless vector store settings
type OpenSearchServerlessConfig struct {
	// CollectionArn is the ARN of the OpenSearch Serverless collection
	CollectionArn string `json:"collectionArn"`

	// VectorIndexName is the name of the vector index
	VectorIndexName string `json:"vectorIndexName"`

	// VectorField is the name of the field containing the vector
	VectorField string `json:"vectorField"`

	// TextField is the name of the field containing the text content
	TextField string `json:"textField"`

	// MetadataField is the name of the field containing metadata
	MetadataField string `json:"metadataField"`
}

// DataSourceSpec defines the configuration for a Knowledge Base data source
type DataSourceSpec struct {
	// Name is the name of the data source
	Name string `json:"name"`

	// Description of the data source
	// +optional
	Description *string `json:"description,omitempty"`

	// DataSourceType defines the type of data source
	// +kubebuilder:validation:Enum=S3;WEB_CRAWLER;CONFLUENCE;SALESFORCE;SHAREPOINT
	DataSourceType string `json:"dataSourceType"`

	// S3Configuration for S3 data sources
	// +optional
	S3Configuration *S3DataSourceConfig `json:"s3Configuration,omitempty"`
}

// S3DataSourceConfig defines S3-specific data source configuration
type S3DataSourceConfig struct {
	// BucketArn is the ARN of the S3 bucket containing the data
	BucketArn string `json:"bucketArn"`

	// InclusionPrefixes are the S3 prefixes to include in the data source
	// +optional
	InclusionPrefixes []string `json:"inclusionPrefixes,omitempty"`

	// ExclusionPrefixes are the S3 prefixes to exclude from the data source
	// +optional
	ExclusionPrefixes []string `json:"exclusionPrefixes,omitempty"`

	// InclusionPatterns are the file patterns to include (e.g., "*.pdf", "*.txt")
	// +optional
	InclusionPatterns []string `json:"inclusionPatterns,omitempty"`

	// ExclusionPatterns are the file patterns to exclude
	// +optional
	ExclusionPatterns []string `json:"exclusionPatterns,omitempty"`
}

// DataSourceStatus represents the status of a data source
type DataSourceStatus struct {
	// Name is the name of the data source
	Name string `json:"name"`

	// DataSourceId is the AWS data source ID
	DataSourceId string `json:"dataSourceId"`

	// Status is the current status of the data source
	Status string `json:"status"`

	// LastIngestionJobId is the ID of the most recent ingestion job
	// +optional
	LastIngestionJobId *string `json:"lastIngestionJobId,omitempty"`

	// LastIngestionJobStatus is the status of the most recent ingestion job
	// +optional
	LastIngestionJobStatus *string `json:"lastIngestionJobStatus,omitempty"`

	// FailureReasons contains any failure reasons for data source operations
	// +optional
	FailureReasons []string `json:"failureReasons,omitempty"`
}

// BedrockResourceStatus defines the observed state of BedrockResource
type BedrockResourceStatus struct {
	// Conditions represent the current status conditions
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase represents the current phase of the resource
	// +kubebuilder:validation:Enum=Creating;Created;Preparing;Prepared;Updating;Deleting;Failed
	// +optional
	Phase string `json:"phase,omitempty"`

	// AgentId is the AWS Agent ID when type is agent
	// +optional
	AgentId *string `json:"agentId,omitempty"`

	// AgentArn is the AWS Agent ARN when type is agent
	// +optional
	AgentArn *string `json:"agentArn,omitempty"`

	// AgentStatus is the AWS Agent status
	// +optional
	AgentStatus *string `json:"agentStatus,omitempty"`

	// AgentVersion is the AWS Agent version
	// +optional
	AgentVersion *string `json:"agentVersion,omitempty"`

	// PreparedAt indicates when the agent was last prepared
	// +optional
	PreparedAt *metav1.Time `json:"preparedAt,omitempty"`

	// InferenceProfileId is the AWS Inference Profile ID when type is inferenceProfile
	// +optional
	InferenceProfileId *string `json:"inferenceProfileId,omitempty"`

	// InferenceProfileArn is the AWS Inference Profile ARN when type is inferenceProfile
	// +optional
	InferenceProfileArn *string `json:"inferenceProfileArn,omitempty"`

	// InferenceProfileStatus is the AWS Inference Profile status
	// +optional
	InferenceProfileStatus *string `json:"inferenceProfileStatus,omitempty"`

	// InferenceProfileType is the AWS Inference Profile type (SYSTEM_DEFINED or APPLICATION)
	// +optional
	InferenceProfileType *string `json:"inferenceProfileType,omitempty"`

	// KnowledgeBase-specific status fields
	// KnowledgeBaseId is the AWS Knowledge Base ID when type is knowledgeBase
	// +optional
	KnowledgeBaseId *string `json:"knowledgeBaseId,omitempty"`

	// KnowledgeBaseArn is the AWS Knowledge Base ARN when type is knowledgeBase
	// +optional
	KnowledgeBaseArn *string `json:"knowledgeBaseArn,omitempty"`

	// KnowledgeBaseStatus is the AWS Knowledge Base status
	// +optional
	KnowledgeBaseStatus *string `json:"knowledgeBaseStatus,omitempty"`

	// DataSourceIds contains the IDs of associated data sources
	// +optional
	DataSourceIds []string `json:"dataSourceIds,omitempty"`

	// DataSources contains detailed information about data sources
	// +optional
	DataSources []DataSourceStatus `json:"dataSources,omitempty"`

	// FailureReasons contains any failure reasons for knowledge base operations
	// +optional
	FailureReasons []string `json:"failureReasons,omitempty"`

	// ResourceId is the generic AWS resource ID
	// +optional
	ResourceId *string `json:"resourceId,omitempty"`

	// ResourceArn is the generic AWS resource ARN
	// +optional
	ResourceArn *string `json:"resourceArn,omitempty"`

	// ObservedGeneration is the generation of the spec that was last processed
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastSyncTime is when the resource was last synchronized with AWS
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// Phase constants for BedrockResource status
const (
	PhaseCreating  = "Creating"
	PhaseCreated   = "Created"
	PhasePreparing = "Preparing"
	PhasePrepared  = "Prepared"
	PhaseUpdating  = "Updating"
	PhaseDeleting  = "Deleting"
	PhaseFailed    = "Failed"
)

// Condition types for BedrockResource
const (
	// ConditionReady indicates whether the resource is ready for use
	ConditionReady = "Ready"

	// ConditionSynced indicates whether the resource is synchronized with AWS
	ConditionSynced = "Synced"

	// ConditionPrepared indicates whether an agent is prepared (agent-specific)
	ConditionPrepared = "Prepared"
)

// Condition reasons
const (
	ReasonReconciling      = "Reconciling"
	ReasonReconcileSuccess = "ReconcileSuccess"
	ReasonReconcileError   = "ReconcileError"
	ReasonCreating         = "Creating"
	ReasonCreated          = "Created"
	ReasonPreparing        = "Preparing"
	ReasonPrepared         = "Prepared"
	ReasonUpdating         = "Updating"
	ReasonDeleting         = "Deleting"
	ReasonFailed           = "Failed"
	ReasonAWSError         = "AWSError"
	ReasonValidationError  = "ValidationError"
)

// DeepCopyObject implements runtime.Object interface
func (br *BedrockResource) DeepCopyObject() runtime.Object {
	return br.DeepCopy()
}

// DeepCopy creates a deep copy of BedrockResource
func (br *BedrockResource) DeepCopy() *BedrockResource {
	if br == nil {
		return nil
	}

	out := &BedrockResource{}
	br.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (br *BedrockResource) DeepCopyInto(out *BedrockResource) {
	*out = *br
	out.TypeMeta = br.TypeMeta
	br.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	br.Spec.DeepCopyInto(&out.Spec)
	br.Status.DeepCopyInto(&out.Status)
}

// DeepCopyObject implements runtime.Object interface for BedrockResourceList
func (brl *BedrockResourceList) DeepCopyObject() runtime.Object {
	return brl.DeepCopy()
}

// DeepCopy creates a deep copy of BedrockResourceList
func (brl *BedrockResourceList) DeepCopy() *BedrockResourceList {
	if brl == nil {
		return nil
	}

	out := &BedrockResourceList{}
	brl.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (brl *BedrockResourceList) DeepCopyInto(out *BedrockResourceList) {
	*out = *brl
	out.TypeMeta = brl.TypeMeta
	brl.ListMeta.DeepCopyInto(&out.ListMeta)
	if brl.Items != nil {
		in, out := &brl.Items, &out.Items
		*out = make([]BedrockResource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto copies all properties of BedrockResourceSpec
func (brs *BedrockResourceSpec) DeepCopyInto(out *BedrockResourceSpec) {
	*out = *brs
	if brs.Agent != nil {
		in, out := &brs.Agent, &out.Agent
		*out = new(AgentSpec)
		(*in).DeepCopyInto(*out)
	}
	if brs.InferenceProfile != nil {
		in, out := &brs.InferenceProfile, &out.InferenceProfile
		*out = new(InferenceProfileSpec)
		(*in).DeepCopyInto(*out)
	}
	if brs.KnowledgeBase != nil {
		in, out := &brs.KnowledgeBase, &out.KnowledgeBase
		*out = new(KnowledgeBaseSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopyInto copies all properties of AgentSpec
func (as *AgentSpec) DeepCopyInto(out *AgentSpec) {
	*out = *as
	if as.Description != nil {
		in, out := &as.Description, &out.Description
		*out = new(string)
		**out = **in
	}
	if as.FoundationModel != nil {
		in, out := &as.FoundationModel, &out.FoundationModel
		*out = new(string)
		**out = **in
	}
	if as.Instruction != nil {
		in, out := &as.Instruction, &out.Instruction
		*out = new(string)
		**out = **in
	}
	if as.AgentResourceRoleArn != nil {
		in, out := &as.AgentResourceRoleArn, &out.AgentResourceRoleArn
		*out = new(string)
		**out = **in
	}
	if as.CustomerEncryptionKeyArn != nil {
		in, out := &as.CustomerEncryptionKeyArn, &out.CustomerEncryptionKeyArn
		*out = new(string)
		**out = **in
	}
	if as.IdleSessionTTLInSeconds != nil {
		in, out := &as.IdleSessionTTLInSeconds, &out.IdleSessionTTLInSeconds
		*out = new(int32)
		**out = **in
	}
	if as.Tags != nil {
		in, out := &as.Tags, &out.Tags
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if as.GuardrailConfiguration != nil {
		in, out := &as.GuardrailConfiguration, &out.GuardrailConfiguration
		*out = new(GuardrailConfiguration)
		(*in).DeepCopyInto(*out)
	}
	if as.MemoryConfiguration != nil {
		in, out := &as.MemoryConfiguration, &out.MemoryConfiguration
		*out = new(MemoryConfiguration)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopyInto copies all properties of GuardrailConfiguration
func (gc *GuardrailConfiguration) DeepCopyInto(out *GuardrailConfiguration) {
	*out = *gc
	if gc.GuardrailId != nil {
		in, out := &gc.GuardrailId, &out.GuardrailId
		*out = new(string)
		**out = **in
	}
	if gc.GuardrailVersion != nil {
		in, out := &gc.GuardrailVersion, &out.GuardrailVersion
		*out = new(string)
		**out = **in
	}
}

// DeepCopyInto copies all properties of MemoryConfiguration
func (mc *MemoryConfiguration) DeepCopyInto(out *MemoryConfiguration) {
	*out = *mc
	if mc.EnabledMemoryTypes != nil {
		in, out := &mc.EnabledMemoryTypes, &out.EnabledMemoryTypes
		*out = make([]MemoryType, len(*in))
		copy(*out, *in)
	}
	if mc.StorageDays != nil {
		in, out := &mc.StorageDays, &out.StorageDays
		*out = new(int32)
		**out = **in
	}
}

// DeepCopyInto copies all properties of InferenceProfileSpec
func (ips *InferenceProfileSpec) DeepCopyInto(out *InferenceProfileSpec) {
	*out = *ips
	if ips.Description != nil {
		in, out := &ips.Description, &out.Description
		*out = new(string)
		**out = **in
	}
	// ModelSource is not a pointer, so simple assignment is fine
	out.ModelSource = ips.ModelSource
	if ips.Tags != nil {
		in, out := &ips.Tags, &out.Tags
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopyInto copies all properties of InferenceProfileModelSource
func (ipms *InferenceProfileModelSource) DeepCopyInto(out *InferenceProfileModelSource) {
	*out = *ipms
}

// DeepCopyInto copies all properties of KnowledgeBaseSpec
func (kbs *KnowledgeBaseSpec) DeepCopyInto(out *KnowledgeBaseSpec) {
	*out = *kbs
	if kbs.Description != nil {
		in, out := &kbs.Description, &out.Description
		*out = new(string)
		**out = **in
	}
	if kbs.OpenSearchServerlessConfiguration != nil {
		in, out := &kbs.OpenSearchServerlessConfiguration, &out.OpenSearchServerlessConfiguration
		*out = new(OpenSearchServerlessConfig)
		(*in).DeepCopyInto(*out)
	}
	if kbs.DataSources != nil {
		in, out := &kbs.DataSources, &out.DataSources
		*out = make([]DataSourceSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if kbs.Tags != nil {
		in, out := &kbs.Tags, &out.Tags
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopyInto copies all properties of OpenSearchServerlessConfig
func (ossc *OpenSearchServerlessConfig) DeepCopyInto(out *OpenSearchServerlessConfig) {
	*out = *ossc
}

// DeepCopyInto copies all properties of DataSourceSpec
func (ds *DataSourceSpec) DeepCopyInto(out *DataSourceSpec) {
	*out = *ds
	if ds.Description != nil {
		in, out := &ds.Description, &out.Description
		*out = new(string)
		**out = **in
	}
	if ds.S3Configuration != nil {
		in, out := &ds.S3Configuration, &out.S3Configuration
		*out = new(S3DataSourceConfig)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopyInto copies all properties of S3DataSourceConfig
func (s3c *S3DataSourceConfig) DeepCopyInto(out *S3DataSourceConfig) {
	*out = *s3c
	if s3c.InclusionPrefixes != nil {
		in, out := &s3c.InclusionPrefixes, &out.InclusionPrefixes
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if s3c.ExclusionPrefixes != nil {
		in, out := &s3c.ExclusionPrefixes, &out.ExclusionPrefixes
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if s3c.InclusionPatterns != nil {
		in, out := &s3c.InclusionPatterns, &out.InclusionPatterns
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if s3c.ExclusionPatterns != nil {
		in, out := &s3c.ExclusionPatterns, &out.ExclusionPatterns
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopyInto copies all properties of DataSourceStatus
func (dss *DataSourceStatus) DeepCopyInto(out *DataSourceStatus) {
	*out = *dss
	if dss.LastIngestionJobId != nil {
		in, out := &dss.LastIngestionJobId, &out.LastIngestionJobId
		*out = new(string)
		**out = **in
	}
	if dss.LastIngestionJobStatus != nil {
		in, out := &dss.LastIngestionJobStatus, &out.LastIngestionJobStatus
		*out = new(string)
		**out = **in
	}
	if dss.FailureReasons != nil {
		in, out := &dss.FailureReasons, &out.FailureReasons
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopyInto copies all properties of BedrockResourceStatus
func (brs *BedrockResourceStatus) DeepCopyInto(out *BedrockResourceStatus) {
	*out = *brs
	if brs.Conditions != nil {
		in, out := &brs.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if brs.AgentId != nil {
		in, out := &brs.AgentId, &out.AgentId
		*out = new(string)
		**out = **in
	}
	if brs.AgentArn != nil {
		in, out := &brs.AgentArn, &out.AgentArn
		*out = new(string)
		**out = **in
	}
	if brs.AgentStatus != nil {
		in, out := &brs.AgentStatus, &out.AgentStatus
		*out = new(string)
		**out = **in
	}
	if brs.AgentVersion != nil {
		in, out := &brs.AgentVersion, &out.AgentVersion
		*out = new(string)
		**out = **in
	}
	if brs.PreparedAt != nil {
		in, out := &brs.PreparedAt, &out.PreparedAt
		*out = (*in).DeepCopy()
	}
	if brs.InferenceProfileId != nil {
		in, out := &brs.InferenceProfileId, &out.InferenceProfileId
		*out = new(string)
		**out = **in
	}
	if brs.InferenceProfileArn != nil {
		in, out := &brs.InferenceProfileArn, &out.InferenceProfileArn
		*out = new(string)
		**out = **in
	}
	if brs.InferenceProfileStatus != nil {
		in, out := &brs.InferenceProfileStatus, &out.InferenceProfileStatus
		*out = new(string)
		**out = **in
	}
	if brs.InferenceProfileType != nil {
		in, out := &brs.InferenceProfileType, &out.InferenceProfileType
		*out = new(string)
		**out = **in
	}
	if brs.KnowledgeBaseId != nil {
		in, out := &brs.KnowledgeBaseId, &out.KnowledgeBaseId
		*out = new(string)
		**out = **in
	}
	if brs.KnowledgeBaseArn != nil {
		in, out := &brs.KnowledgeBaseArn, &out.KnowledgeBaseArn
		*out = new(string)
		**out = **in
	}
	if brs.KnowledgeBaseStatus != nil {
		in, out := &brs.KnowledgeBaseStatus, &out.KnowledgeBaseStatus
		*out = new(string)
		**out = **in
	}
	if brs.DataSourceIds != nil {
		in, out := &brs.DataSourceIds, &out.DataSourceIds
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if brs.DataSources != nil {
		in, out := &brs.DataSources, &out.DataSources
		*out = make([]DataSourceStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if brs.FailureReasons != nil {
		in, out := &brs.FailureReasons, &out.FailureReasons
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if brs.ResourceId != nil {
		in, out := &brs.ResourceId, &out.ResourceId
		*out = new(string)
		**out = **in
	}
	if brs.ResourceArn != nil {
		in, out := &brs.ResourceArn, &out.ResourceArn
		*out = new(string)
		**out = **in
	}
	if brs.LastSyncTime != nil {
		in, out := &brs.LastSyncTime, &out.LastSyncTime
		*out = (*in).DeepCopy()
	}
}
