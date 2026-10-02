#!/bin/bash

# Quick cleanup script for Knowledge Base testing resources
# This script removes all AWS resources created during KB testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGION="us-east-1"
KB_ROLE_NAME="AmazonBedrockExecutionRoleForKnowledgeBase_TestRole"
COLLECTION_NAME="kb-collection"
S3_BUCKET_NAME="bedrock-kb-test-data-${ACCOUNT_ID}"
POLICY_NAME="BedrockKnowledgeBaseTestPolicy"

echo -e "${RED}🧹 Cleaning up Knowledge Base testing resources...${NC}"
echo "Account: $ACCOUNT_ID"
echo "Region: $REGION"
echo ""
echo -e "${YELLOW}⚠️  This will delete all AWS resources created for KB testing!${NC}"
read -p "Are you sure you want to continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled."
    exit 0
fi

echo -e "${GREEN}Starting cleanup...${NC}"

# 1. Stop any running controller
echo "Step 1: Stopping controller..."
pkill -f controller-local 2>/dev/null && echo "Controller stopped" || echo "No controller running"

# 2. Remove Kubernetes finalizers and delete resources
echo "Step 2: Cleaning up Kubernetes resources..."
if kubectl get bedrockresource test-knowledge-base >/dev/null 2>&1; then
    echo "Removing finalizers from test-knowledge-base..."
    kubectl patch bedrockresource test-knowledge-base -p '{"metadata":{"finalizers":null}}' --type=merge 2>/dev/null || echo "Finalizer removal failed or not needed"
    sleep 2
    if kubectl get bedrockresource test-knowledge-base >/dev/null 2>&1; then
        echo "Resource still exists after finalizer removal"
    else
        echo "Resource successfully deleted"
    fi
else
    echo "No test-knowledge-base resource found"
fi

# 3. Delete any created Knowledge Bases in AWS
echo "Step 3: Checking for Knowledge Bases in AWS..."
KB_LIST=$(aws bedrock-agent list-knowledge-bases --region $REGION --query 'knowledgeBaseSummaries[?contains(name, `Test`)].[knowledgeBaseId,name]' --output text 2>/dev/null || echo "")
if [ -n "$KB_LIST" ]; then
    echo "Found test Knowledge Bases:"
    echo "$KB_LIST"
    while read -r KB_ID KB_NAME; do
        if [ -n "$KB_ID" ]; then
            echo "Deleting Knowledge Base: $KB_NAME ($KB_ID)"
            aws bedrock-agent delete-knowledge-base --knowledge-base-id "$KB_ID" --region $REGION 2>/dev/null || echo "Failed to delete KB $KB_ID"
        fi
    done <<< "$KB_LIST"
else
    echo "No test Knowledge Bases found in AWS"
fi

# 4. Delete OpenSearch Serverless collection
echo "Step 4: Deleting OpenSearch Serverless collection..."
if aws opensearchserverless list-collections --collection-filters name="$COLLECTION_NAME" | grep -q "$COLLECTION_NAME"; then
    echo "Deleting collection: $COLLECTION_NAME"
    COLLECTION_ID=$(aws opensearchserverless list-collections --collection-filters name="$COLLECTION_NAME" --query 'collectionSummaries[0].id' --output text)
    aws opensearchserverless delete-collection --id "$COLLECTION_ID" 2>/dev/null && echo "Collection deletion initiated" || echo "Failed to delete collection"
    
    # Wait for deletion to complete
    echo "Waiting for collection deletion to complete..."
    while aws opensearchserverless list-collections --collection-filters name="$COLLECTION_NAME" 2>/dev/null | grep -q "$COLLECTION_NAME"; do
        echo "Collection still exists, waiting..."
        sleep 10
    done
    echo "Collection deleted successfully"
else
    echo "Collection $COLLECTION_NAME not found"
fi

# 5. Delete OpenSearch Serverless policies
echo "Step 5: Deleting OpenSearch Serverless policies..."

# Data access policy
if aws opensearchserverless get-access-policy --name "kb-collection-access-policy" --type data >/dev/null 2>&1; then
    echo "Deleting data access policy..."
    aws opensearchserverless delete-access-policy --name "kb-collection-access-policy" --type data 2>/dev/null && echo "Data access policy deleted" || echo "Failed to delete data access policy"
else
    echo "Data access policy not found"
fi

# Network policy
if aws opensearchserverless get-security-policy --name "kb-collection-network-policy" --type network >/dev/null 2>&1; then
    echo "Deleting network policy..."
    aws opensearchserverless delete-security-policy --name "kb-collection-network-policy" --type network 2>/dev/null && echo "Network policy deleted" || echo "Failed to delete network policy"
else
    echo "Network policy not found"
fi

# Encryption policy
if aws opensearchserverless get-security-policy --name "kb-collection-encryption-policy" --type encryption >/dev/null 2>&1; then
    echo "Deleting encryption policy..."
    aws opensearchserverless delete-security-policy --name "kb-collection-encryption-policy" --type encryption 2>/dev/null && echo "Encryption policy deleted" || echo "Failed to delete encryption policy"
else
    echo "Encryption policy not found"
fi

# 6. Delete S3 bucket and contents
echo "Step 6: Deleting S3 bucket and contents..."
if aws s3 ls "s3://$S3_BUCKET_NAME" >/dev/null 2>&1; then
    echo "Deleting S3 bucket contents..."
    aws s3 rm "s3://$S3_BUCKET_NAME" --recursive 2>/dev/null && echo "S3 contents deleted" || echo "Failed to delete S3 contents"
    
    echo "Deleting S3 bucket..."
    aws s3 rb "s3://$S3_BUCKET_NAME" 2>/dev/null && echo "S3 bucket deleted" || echo "Failed to delete S3 bucket"
else
    echo "S3 bucket $S3_BUCKET_NAME not found"
fi

# 7. Delete IAM role and policy
echo "Step 7: Deleting IAM role and policy..."

# Delete inline policy
if aws iam get-role-policy --role-name "$KB_ROLE_NAME" --policy-name "$POLICY_NAME" >/dev/null 2>&1; then
    echo "Deleting inline policy from role..."
    aws iam delete-role-policy --role-name "$KB_ROLE_NAME" --policy-name "$POLICY_NAME" 2>/dev/null && echo "Inline policy deleted" || echo "Failed to delete inline policy"
else
    echo "Inline policy not found"
fi

# Delete IAM role
if aws iam get-role --role-name "$KB_ROLE_NAME" >/dev/null 2>&1; then
    echo "Deleting IAM role..."
    aws iam delete-role --role-name "$KB_ROLE_NAME" 2>/dev/null && echo "IAM role deleted" || echo "Failed to delete IAM role"
else
    echo "IAM role not found"
fi

# 8. Clean up local files
echo "Step 8: Cleaning up local files..."
rm -f kb-test-config.yaml 2>/dev/null && echo "Removed kb-test-config.yaml" || echo "kb-test-config.yaml not found"
rm -f /tmp/fix-data-access-policy.json 2>/dev/null || true

echo ""
echo -e "${GREEN}✅ Cleanup completed!${NC}"
echo ""
echo "Summary of actions taken:"
echo "  - Stopped controller process"
echo "  - Removed Kubernetes finalizers and resources"
echo "  - Deleted any test Knowledge Bases from AWS"
echo "  - Deleted OpenSearch Serverless collection and policies"
echo "  - Deleted S3 bucket and contents"
echo "  - Deleted IAM role and policies"
echo "  - Cleaned up local configuration files"
echo ""
echo -e "${YELLOW}Note: Some resources may take a few minutes to fully delete.${NC}"
echo "Your AWS account should now be clean of all Knowledge Base testing resources."
