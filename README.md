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
    foundationModel: "anthropic.claude-v2"
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

### AWS Credentials

The controller supports multiple AWS credential sources:
- Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- IAM roles for service accounts (IRSA)
- Instance profiles
- Shared credentials file

### Environment Variables

- `AWS_REGION`: AWS region for Bedrock services (default: us-east-1)
- `KUBECONFIG`: Path to kubeconfig file
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

## Project Structure

```
.
├── controller/          # Controller implementation
│   ├── main.go         # Main entry point
│   ├── reconcile.go    # Reconciliation logic
│   ├── bedrock_client.go # AWS Bedrock client wrapper
│   └── types.go        # Kubernetes types
├── crd/                # Custom Resource Definitions
├── cr/                 # Example Custom Resources
├── references/         # Read-only reference code (gitignored)
├── tests/              # Unit tests (gitignored)
├── Makefile           # Build and development targets
└── README.md          # This file
```

## Contributing

1. Follow the iterative development approach
2. Ensure unit tests pass before advancing
3. Update CRD and example CRs in lockstep with code changes
4. Use conventional commits (feat:, fix:, chore:, etc.)

## License

[Add your license here]
