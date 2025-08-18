# Knowledge Base AWS Testing - Complete Setup Documentation

This document tracks all AWS resources created during Knowledge Base integration testing and provides step-by-step instructions for setup and cleanup.

## 🗂️ AWS Resources Created

### Account Information
- **AWS Account ID**: `REPLACE_ME_AWS_ACCOUNT_ID`
- **Region**: `us-east-1`
- **Testing Date**: `2025-08-15`

### 1. IAM Role for Knowledge Base Execution

**Resource Created:**
```
Role Name: AmazonBedrockExecutionRoleForKnowledgeBase_TestRole
Role ARN: arn:aws:iam::REPLACE_ME_AWS_ACCOUNT_ID:role/AmazonBedrockExecutionRoleForKnowledgeBase_TestRole
```

**Trust Policy:**
```json
{
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
}
```

**Attached Policy:**
```
Policy Name: BedrockKnowledgeBaseTestPolicy
Policy Type: Inline Policy
```

**Policy Document:**
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "bedrock:InvokeModel",
                "bedrock:Retrieve",
                "bedrock:RetrieveAndGenerate"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "aoss:APIAccessAll"
            ],
            "Resource": "arn:aws:aoss:us-east-1:REPLACE_ME_AWS_ACCOUNT_ID:collection/*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::bedrock-kb-test-data-REPLACE_ME_AWS_ACCOUNT_ID",
                "arn:aws:s3:::bedrock-kb-test-data-REPLACE_ME_AWS_ACCOUNT_ID/*"
            ]
        }
    ]
}
```

### 2. S3 Bucket for Data Source

**Resource Created:**
```
Bucket Name: bedrock-kb-test-data-REPLACE_ME_AWS_ACCOUNT_ID
Region: us-east-1
```

**Contents:**
```
s3://bedrock-kb-test-data-REPLACE_ME_AWS_ACCOUNT_ID/docs/product-info.txt
s3://bedrock-kb-test-data-REPLACE_ME_AWS_ACCOUNT_ID/docs/faq.txt
```

### 3. OpenSearch Serverless Collection

**Resource Created:**
```
Collection Name: kb-collection
Collection ID: REPLACE_ME_COLLECTION_ID
Collection ARN: arn:aws:aoss:us-east-1:REPLACE_ME_AWS_ACCOUNT_ID:collection/REPLACE_ME_COLLECTION_ID
Type: VECTORSEARCH
Endpoint: https://REPLACE_ME_COLLECTION_ID.us-east-1.aoss.amazonaws.com
Dashboard: https://REPLACE_ME_COLLECTION_ID.us-east-1.aoss.amazonaws.com/_dashboards
```

### 4. OpenSearch Serverless Security Policies

#### Encryption Policy
```
Policy Name: kb-collection-encryption-policy
Policy Type: encryption
```

**Policy Document:**
```json
{
    "Rules": [
        {
            "Resource": ["collection/kb-collection"],
            "ResourceType": "collection"
        }
    ],
    "AWSOwnedKey": true
}
```

#### Network Policy
```
Policy Name: kb-collection-network-policy
Policy Type: network
```

**Policy Document:**
```json
[
    {
        "Rules": [
            {
                "Resource": ["collection/kb-collection"],
                "ResourceType": "collection"
            }
        ],
        "AllowFromPublic": true
    }
]
```

#### Data Access Policy
```
Policy Name: kb-collection-access-policy
Policy Type: data
```

**Policy Document (Updated):**
```json
[
    {
        "Rules": [
            {
                "Resource": ["collection/kb-collection"],
                "Permission": [
                    "aoss:CreateCollectionItems",
                    "aoss:DeleteCollectionItems", 
                    "aoss:UpdateCollectionItems",
                    "aoss:DescribeCollectionItems"
                ],
                "ResourceType": "collection"
            },
            {
                "Resource": ["index/kb-collection/*"],
                "Permission": [
                    "aoss:CreateIndex",
                    "aoss:DeleteIndex",
                    "aoss:UpdateIndex",
                    "aoss:DescribeIndex",
                    "aoss:ReadDocument",
                    "aoss:WriteDocument"
                ],
                "ResourceType": "index"
            }
        ],
        "Principal": [
            "arn:aws:iam::REPLACE_ME_AWS_ACCOUNT_ID:role/AmazonBedrockExecutionRoleForKnowledgeBase_TestRole",
            "arn:aws:iam::REPLACE_ME_AWS_ACCOUNT_ID:root"
        ],
        "Description": "Data access policy for Knowledge Base testing with Bedrock service access"
    }
]
```

### 5. Kubernetes Resources

**CRD Applied:**
```
File: crd/bedrockresource.yaml
Status: Applied and configured with Knowledge Base support
```

**Test Resource (Deleted due to finalizer issue):**
```
Name: test-knowledge-base
Namespace: default
Type: BedrockResource (knowledgeBase)
Status: Deleted (was stuck with finalizer)
```

---

## 🔧 Setup Instructions for Other Systems

### Prerequisites
1. **AWS CLI installed and configured** with appropriate credentials
2. **kubectl installed** with access to a Kubernetes cluster  
3. **jq installed** for JSON processing
4. **Bedrock Foundation Models enabled** in your AWS account for `us-east-1`

### Step-by-Step Setup

#### 1. Clone and Build the Controller
```bash
git clone <repository-url>
cd bedrock-manager-controller
make build-local
```

#### 2. Run the AWS Setup Script
```bash
# Make the script executable
chmod +x scripts/setup-kb-aws-resources.sh

# Run the setup (will create all AWS resources)
./scripts/setup-kb-aws-resources.sh
```

**This script creates:**
- IAM execution role for Knowledge Base
- S3 bucket with sample documents
- OpenSearch Serverless collection with security policies
- Data access policies

#### 3. Apply Kubernetes Resources
```bash
# Apply the CRD
kubectl apply -f crd/bedrockresource.yaml

# Apply the generated Knowledge Base test configuration
kubectl apply -f kb-test-config.yaml
```

#### 4. Start the Controller
```bash
# Set AWS region and start controller
AWS_REGION=us-east-1 ./bin/controller-local
```

#### 5. Monitor the Knowledge Base Creation
```bash
# Check resource status
kubectl get bedrockresource test-knowledge-base -o yaml

# Watch for changes
kubectl get bedrockresource test-knowledge-base -w
```

---

## 🧹 Complete Cleanup Instructions

### 1. Stop the Controller
```bash
# If running in background
pkill -f controller-local

# If running in foreground, use Ctrl+C
```

### 2. Delete Kubernetes Resources
```bash
# Delete the test Knowledge Base resource
kubectl delete bedrockresource test-knowledge-base 2>/dev/null || echo "Already deleted"

# If stuck with finalizer, remove it manually:
kubectl patch bedrockresource test-knowledge-base -p '{"metadata":{"finalizers":null}}' --type=merge 2>/dev/null || echo "Already deleted"
```

### 3. Delete AWS Resources (In Order)

#### Delete Any Created Knowledge Bases
```bash
# List knowledge bases to check
aws bedrock-agent list-knowledge-bases --region us-east-1

# Delete any created knowledge bases (replace KB_ID with actual ID)
# aws bedrock-agent delete-knowledge-base --knowledge-base-id KB_ID --region us-east-1
```

#### Delete OpenSearch Serverless Resources
```bash
# Delete the collection
aws opensearchserverless delete-collection --id REPLACE_ME_COLLECTION_ID

# Delete data access policy
aws opensearchserverless delete-access-policy --name kb-collection-access-policy --type data

# Delete network policy  
aws opensearchserverless delete-security-policy --name kb-collection-network-policy --type network

# Delete encryption policy
aws opensearchserverless delete-security-policy --name kb-collection-encryption-policy --type encryption
```

#### Delete S3 Resources
```bash
# Delete S3 bucket contents and bucket
aws s3 rm s3://bedrock-kb-test-data-REPLACE_ME_AWS_ACCOUNT_ID --recursive
aws s3 rb s3://bedrock-kb-test-data-REPLACE_ME_AWS_ACCOUNT_ID
```

#### Delete IAM Resources
```bash
# Delete the inline policy from the role
aws iam delete-role-policy --role-name AmazonBedrockExecutionRoleForKnowledgeBase_TestRole --policy-name BedrockKnowledgeBaseTestPolicy

# Delete the IAM role
aws iam delete-role --role-name AmazonBedrockExecutionRoleForKnowledgeBase_TestRole
```

---

## 🚨 Current Status & Issues

### What's Working
- ✅ All AWS resources created successfully
- ✅ Controller builds and starts correctly  
- ✅ CRD applied successfully
- ✅ Kubernetes resource creation works
- ✅ Finalizer handling works (manually tested)

### Current Issue
- ❌ **OpenSearch Access**: Bedrock service getting `403 Forbidden` when accessing OpenSearch collection
- **Error**: `ValidationException: The knowledge base storage configuration provided is invalid... Request failed: [security_exception] 403 Forbidden`

### Next Steps to Resolve
1. **Verify OpenSearch permissions**: The data access policy was updated to include root account access
2. **Test vector index creation**: May need to pre-create the vector index in OpenSearch
3. **Check Bedrock service permissions**: Verify Bedrock has proper access to the collection

### Files Generated
- `kb-test-config.yaml` - Complete Knowledge Base test configuration
- `scripts/setup-kb-aws-resources.sh` - Complete setup script
- `scripts/create-vector-index.sh` - Vector index creation script (if needed)

---

## 📋 Quick Cleanup Script

For convenience, here's a quick cleanup script:

```bash
#!/bin/bash
# quick-cleanup.sh

echo "Cleaning up Knowledge Base testing resources..."

# Stop controller
pkill -f controller-local 2>/dev/null || true

# Remove finalizers and delete K8s resources
kubectl patch bedrockresource test-knowledge-base -p '{"metadata":{"finalizers":null}}' --type=merge 2>/dev/null || true

# Delete AWS resources
aws opensearchserverless delete-collection --id REPLACE_ME_COLLECTION_ID 2>/dev/null || true
aws opensearchserverless delete-access-policy --name kb-collection-access-policy --type data 2>/dev/null || true
aws opensearchserverless delete-security-policy --name kb-collection-network-policy --type network 2>/dev/null || true
aws opensearchserverless delete-security-policy --name kb-collection-encryption-policy --type encryption 2>/dev/null || true

# Clean S3
aws s3 rm s3://bedrock-kb-test-data-REPLACE_ME_AWS_ACCOUNT_ID --recursive 2>/dev/null || true
aws s3 rb s3://bedrock-kb-test-data-REPLACE_ME_AWS_ACCOUNT_ID 2>/dev/null || true

# Clean IAM
aws iam delete-role-policy --role-name AmazonBedrockExecutionRoleForKnowledgeBase_TestRole --policy-name BedrockKnowledgeBaseTestPolicy 2>/dev/null || true
aws iam delete-role --role-name AmazonBedrockExecutionRoleForKnowledgeBase_TestRole 2>/dev/null || true

echo "Cleanup completed!"
```

---

*This documentation will be updated as we resolve the current OpenSearch access issue and complete the testing.*
