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

## Inference Profiles

Inference Profiles enable cost tracking and metrics collection for foundation models. They allow you to:

- Track usage and costs for specific models or teams
- Create application-specific inference configurations
- Copy from existing foundation models or system-defined profiles

### Basic Inference Profile

```yaml
apiVersion: bedrock.aws.example.com/v1
kind: BedrockResource
metadata:
  name: my-cost-tracker
spec:
  type: inferenceProfile
  inferenceProfile:
    name: "MyCostTracker"
    description: "Track usage for our application"
    modelSource:
      copyFrom: "arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-micro-v1:0"
    tags:
      Team: "ai-platform"
      Purpose: "cost-tracking"
```

### Available Foundation Models

Popular foundation model ARNs for `copyFrom`:

**Amazon Nova Models:**
- `arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-micro-v1:0`
- `arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-lite-v1:0`
- `arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-pro-v1:0`

**Anthropic Claude Models:**
- `arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-5-sonnet-20241022-v2:0`
- `arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-5-haiku-20241022-v1:0`

**System-Defined Inference Profiles:**
- `arn:aws:bedrock:us-east-1::inference-profile/anthropic.claude-3-5-sonnet-20241022-v2:0-cross-region`

### Model Source Options

**copyFrom** can reference:
1. **Foundation Models**: Direct model ARNs for single-region usage
2. **System-Defined Inference Profiles**: Pre-configured cross-region profiles for high availability

### Use Cases

1. **Cost Tracking**: Monitor spending per team/application
2. **Performance Analysis**: Collect metrics for specific use cases  
3. **Multi-Region Deployments**: Use cross-region inference profiles for HA
4. **Development vs Production**: Separate profiles for different environments

## Knowledge Bases

Knowledge Bases enable Retrieval-Augmented Generation (RAG) by storing and indexing documents as vector embeddings. They integrate with vector databases and can be used with Bedrock Agents for enhanced AI capabilities.

### Basic Knowledge Base

```yaml
apiVersion: bedrock.aws.example.com/v1
kind: BedrockResource
metadata:
  name: product-knowledge-base
spec:
  type: knowledgeBase
  knowledgeBase:
    name: "ProductKnowledgeBase"
    description: "Knowledge base containing product information and FAQs"
    roleArn: "arn:aws:iam::ACCOUNT:role/AmazonBedrockExecutionRoleForKnowledgeBase_TestRole"
    embeddingModelArn: "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1"
    vectorStoreType: "OPENSEARCH_SERVERLESS"
    opensearchServerlessConfiguration:
      collectionArn: "arn:aws:aoss:us-east-1:ACCOUNT:collection/kb-collection"
      vectorIndexName: "vector-index"
      vectorField: "vector"
      textField: "text"
      metadataField: "metadata"
```

### Available Embedding Models

Common embedding model ARNs for `embeddingModelArn`:

**Amazon Titan Embedding Models:**
- `arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1`
- `arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v2:0`
- `arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-g1-text-02`

**Cohere Embedding Models:**
- `arn:aws:bedrock:us-east-1::foundation-model/cohere.embed-english-v3`
- `arn:aws:bedrock:us-east-1::foundation-model/cohere.embed-multilingual-v3`

### Vector Store Types

Currently supported vector store types:

1. **OPENSEARCH_SERVERLESS**: Amazon OpenSearch Serverless (recommended)
   - Requires: `collectionArn`, `vectorIndexName`, field mappings
   - Fully managed, auto-scaling
   - Best for most use cases

2. **PINECONE**: Pinecone vector database
   - Requires: `connectionString`, `credentialsSecretArn`, field mappings
   - Third-party managed service

3. **REDIS_ENTERPRISE_CLOUD**: Redis Enterprise Cloud
   - Requires: `endpoint`, `vectorIndexName`, `credentialsSecretArn`, field mappings
   - High-performance option

### IAM Role Requirements

The `roleArn` must have permissions to:

1. **OpenSearch Serverless** (if using):
   - `aoss:APIAccessAll` on the collection
   - `aoss:DashboardsAccessAll` for console access

2. **Bedrock**:
   - `bedrock:InvokeModel` for the embedding model
   - `bedrock:GetFoundationModel` for model details

3. **S3** (for data sources):
   - `s3:GetObject` and `s3:ListBucket` on data source buckets

### Field Mapping

Vector stores require field mappings to organize data:

- **vectorField**: Stores the embedding vectors
- **textField**: Stores the original text content  
- **metadataField**: Stores document metadata (source, chunk info, etc.)

### Use Cases

1. **Document Search**: Enable semantic search across enterprise documents
2. **FAQ Systems**: Build intelligent FAQ bots with contextual responses
3. **Customer Support**: Provide agents with relevant knowledge base articles
4. **Code Documentation**: Search and reference code documentation and examples
5. **Product Catalogs**: Enhance product recommendations with semantic search
