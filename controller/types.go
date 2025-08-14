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

	// Additional fields will be added in future iterations
}

// KnowledgeBaseSpec defines the configuration for a Bedrock Knowledge Base
type KnowledgeBaseSpec struct {
	// Name for the knowledge base
	Name string `json:"name"`

	// Description of the knowledge base
	// +optional
	Description *string `json:"description,omitempty"`

	// Additional fields will be added in future iterations
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
	ReasonReconciling     = "Reconciling"
	ReasonReconcileError  = "ReconcileError"
	ReasonCreating        = "Creating"
	ReasonCreated         = "Created"
	ReasonPreparing       = "Preparing"
	ReasonPrepared        = "Prepared"
	ReasonUpdating        = "Updating"
	ReasonDeleting        = "Deleting"
	ReasonFailed          = "Failed"
	ReasonAWSError        = "AWSError"
	ReasonValidationError = "ValidationError"
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
}

// DeepCopyInto copies all properties of KnowledgeBaseSpec
func (kbs *KnowledgeBaseSpec) DeepCopyInto(out *KnowledgeBaseSpec) {
	*out = *kbs
	if kbs.Description != nil {
		in, out := &kbs.Description, &out.Description
		*out = new(string)
		**out = **in
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
