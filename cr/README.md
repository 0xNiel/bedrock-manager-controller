# Custom Resource Examples

This directory contains example Custom Resources (CRs) for the Unified AWS Bedrock Controller.

## Agent Examples

### Basic Agent
**File**: `bedrockagent-sample.yaml`

A simple customer service agent with:
- Basic configuration
- Auto-prepare enabled
- Simple instruction set
- Basic tags

### Advanced Agent  
**File**: `bedrockagent-advanced.yaml`

An advanced agent with:
- Custom IAM role and KMS encryption
- Guardrail configuration
- Memory configuration with session summaries
- Extended session timeout
- Compliance tags

## Inference Profile Example

**File**: `inference-profile-sample.yaml`

A placeholder for inference profile configuration. Full implementation will be added in future iterations.

## Knowledge Base Example

**File**: `knowledge-base-sample.yaml` 

A placeholder for knowledge base configuration. Full implementation will be added in future iterations.

## Usage

1. **Apply the CRD first**:
   ```bash
   kubectl apply -f crd/bedrockresource.yaml
   ```

2. **Create a resource**:
   ```bash
   kubectl apply -f cr/bedrockagent-sample.yaml
   ```

3. **Check status**:
   ```bash
   kubectl get bedrockresources
   kubectl describe bedrockresource customer-service-agent
   ```

4. **View logs**:
   ```bash
   kubectl logs -f deployment/bedrock-controller
   ```

## Configuration Notes

### Agent Configuration

- **foundationModel**: Use valid Bedrock model IDs. Examples:
  - `amazon.nova-micro-v1:0` (cost-effective, fast)
  - `amazon.nova-lite-v1:0` (balanced performance)  
  - `amazon.nova-pro-v1:0` (highest capability)
  - `anthropic.claude-3-haiku-20240307-v1:0` (legacy, faster)
  - `anthropic.claude-3-sonnet-20240229-v1:0` (legacy, balanced)
  - `anthropic.claude-3-opus-20240229-v1:0` (legacy, highest capability)

- **autoPrepare**: When `true`, the controller automatically calls PrepareAgent after creation or updates

- **idleSessionTTLInSeconds**: Session timeout (60-3600 seconds)

- **agentResourceRoleArn**: IAM role that the agent assumes (e.g., `arn:aws:iam::REPLACE_ME_AWS_ACCOUNT_ID:role/AmazonBedrockExecutionRoleForAgents_TestRole`). Must have permissions for:
  - Bedrock model invocation
  - Any action groups or knowledge bases
  - CloudWatch logging

### Security Considerations

- Use IAM roles with least privilege principles
- Enable KMS encryption for sensitive agents
- Configure guardrails for production agents
- Set appropriate session timeouts
- Use proper tagging for compliance and cost tracking

### Memory Configuration

- **SESSION_SUMMARY**: Enables session summary memory
- **storageDays**: How long to retain memory (1-999 days)

### Tags

Common tag patterns:
- **Environment**: `dev`, `staging`, `production`
- **Team**: Owning team identifier
- **Purpose**: Use case or function
- **Compliance**: Compliance requirements if any
