# Unified AWS Bedrock Controller

A unified Kubernetes controller for managing AWS Bedrock resources including Agents, Inference Profiles, and Knowledge Bases.

## Overview

This controller provides a single, unified approach to managing AWS Bedrock resources in Kubernetes. Instead of deploying multiple controllers, this solution manages all Bedrock resources through a single controller deployment per tenant.

### Supported Resources

- **Agents**: Bedrock conversational AI agents with auto-prepare capability
- **Inference Profiles**: Custom inference configurations 
- **Knowledge Bases**: Vector databases for retrieval-augmented generation (RAG)

### Optional Agent Features (Planned)
- Action Groups
- Flows  
- Guardrails
- Aliases
- Multi-Agent Collaboration
- Versions

## Architecture

- **Single Controller**: One controller manages all Bedrock resource types
- **Unified CRD**: `BedrockResource` CRD supports all resource types
- **Separate CRs**: Create individual Custom Resources for each resource type
- **AWS SDK Go v2**: Built with pinned AWS SDK for Go v2
- **Client-go**: Uses client-go patterns (no Kubebuilder/Operator SDK)

## Quick Start

### Prerequisites

- Go 1.22+
- Kubernetes cluster
- AWS credentials configured (IAM roles, environment variables, or IRSA)
- Required AWS permissions for Bedrock services

### Installation

1. **Clone and build**:
   ```bash
   git clone <repo-url>
   cd bedrock-manager-controller
   make dev-setup
   make build-local
   ```

2. **Apply CRDs**:
   ```bash
   kubectl apply -f crd/
   ```

3. **Deploy controller**:
   ```bash
   # Build and deploy
   make build-deploy
   make docker-deploy
   # Apply deployment manifests (when created)
   ```

### Usage Examples

#### Create a Bedrock Agent
```yaml
apiVersion: bedrock.aws.example.com/v1
kind: BedrockResource
metadata:
  name: my-agent
  namespace: default
spec:
  type: agent
  agent:
    name: "CustomerServiceAgent"
    description: "AI agent for customer service"
    foundationModel: "amazon.nova-micro-v1:0"
    autoPrepare: true
    instruction: "You are a helpful customer service agent."
```

#### Create an Inference Profile
```yaml
apiVersion: bedrock.aws.example.com/v1
kind: BedrockResource
metadata:
  name: my-inference-profile  
  namespace: default
spec:
  type: inferenceProfile
  inferenceProfile:
    name: "CustomInferenceProfile"
    description: "Custom inference configuration"
    # Additional inference profile configuration
```

#### Create a Knowledge Base
```yaml
apiVersion: bedrock.aws.example.com/v1
kind: BedrockResource
metadata:
  name: my-knowledge-base
  namespace: default
spec:
  type: knowledgeBase
  knowledgeBase:
    name: "ProductKnowledgeBase"
    description: "Knowledge base for product information"
    # Additional knowledge base configuration
```

## Development

### Building

```bash
# Local development (macOS)
make build-local

# Production deployment (Linux)
make build-deploy

# Docker images
make docker-local    # for local platform
make docker-deploy   # for deployment
```

### Testing

```bash
# Run unit tests
make test

# Run linting
make lint

# Run all checks
make dev-check
```

### Validation

```bash
# Validate CRD files
make crd-validate

# Check type synchronization
make cr-sync
```

## Configuration

### AWS IAM Roles Setup

**IMPORTANT**: This controller requires **TWO separate IAM roles**:

#### 1. Controller IAM Role (for Kubernetes controller to call AWS APIs)

**Purpose**: Allows the controller pod to make AWS Bedrock API calls

**For EKS with IRSA** (Recommended):
```bash
# 1. Create IAM role for the controller
aws iam create-role --role-name BedrockControllerRole --assume-role-policy-document '{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::REPLACE_ME_AWS_ACCOUNT_ID:oidc-provider/oidc.eks.YOUR_REGION.amazonaws.com/id/YOUR_CLUSTER_ID"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "oidc.eks.YOUR_REGION.amazonaws.com/id/YOUR_CLUSTER_ID:sub": "system:serviceaccount:bedrock-system:bedrock-controller",
          "oidc.eks.YOUR_REGION.amazonaws.com/id/YOUR_CLUSTER_ID:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}'

# 2. Attach Bedrock permissions to controller role
aws iam attach-role-policy --role-name BedrockControllerRole --policy-arn arn:aws:iam::aws:policy/AmazonBedrockFullAccess

# 3. Update deploy/serviceaccount.yaml with the role ARN:
#    eks.amazonaws.com/role-arn: arn:aws:iam::REPLACE_ME_AWS_ACCOUNT_ID:role/BedrockControllerRole
```

**For local development or non-EKS**:
- AWS credentials via `~/.aws/credentials`
- Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- EC2 instance profiles

#### 2. Agent Execution Role (for Bedrock agents to assume when running)

**Purpose**: The role that Bedrock agents assume to access other AWS services

```bash
# 1. Create IAM role for Bedrock agents
aws iam create-role --role-name AmazonBedrockExecutionRoleForAgents --assume-role-policy-document '{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "bedrock.amazonaws.com"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "aws:SourceAccount": "REPLACE_ME_AWS_ACCOUNT_ID"
        }
      }
    }
  ]
}'

# 2. Attach basic Bedrock permissions
aws iam attach-role-policy --role-name AmazonBedrockExecutionRoleForAgents --policy-arn arn:aws:iam::aws:policy/AmazonBedrockFullAccess

# 3. Use this role ARN in your BedrockResource CRs:
#    spec.agent.agentResourceRoleArn: arn:aws:iam::REPLACE_ME_AWS_ACCOUNT_ID:role/AmazonBedrockExecutionRoleForAgents
```

**Replace placeholders**:
- `REPLACE_ME_AWS_ACCOUNT_ID`: Your AWS account ID
- `YOUR_REGION`: Your AWS region (e.g., us-east-1)
- `YOUR_CLUSTER_ID`: Your EKS cluster OIDC issuer ID

### Environment Variables

- `AWS_REGION`: AWS region for Bedrock services (default: us-east-1)
- `KUBECONFIG`: Path to kubeconfig file
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

## Project Structure

```
.
├── controller/          # Controller implementation
│   ├── reconcile.go    # Reconciliation logic
│   ├── bedrock_client.go # AWS Bedrock client wrapper
│   └── types.go        # Kubernetes types
├── crd/                # Custom Resource Definitions
├── cr/                 # Example Custom Resources
├── deploy/             # Kubernetes deployment manifests
│   ├── deployment.yaml # Controller deployment
│   ├── rbac.yaml      # RBAC configuration
│   ├── serviceaccount.yaml # Service account with IRSA
│   └── README.md      # Deployment guide
├── references/         # Read-only reference code (gitignored)
├── tests/              # Unit tests (gitignored)
├── main.go            # Main entry point
├── Dockerfile         # Container image definition
├── Makefile           # Build and development targets
└── README.md          # This file
```

## Contributing

1. Follow the iterative development approach
2. Ensure unit tests pass before advancing
3. Update CRD and example CRs in lockstep with code changes
4. Use conventional commits (feat:, fix:, chore:, etc.)

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
