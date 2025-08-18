#!/bin/bash

# Setup AWS resources for Knowledge Base testing
# This script creates all prerequisites needed for Knowledge Base integration tests
# Updated to handle complete end-to-end setup with proper dependencies

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGION="us-east-1"
KB_ROLE_NAME="AmazonBedrockExecutionRoleForKnowledgeBase_TestRole"
COLLECTION_NAME="kb-collection"
S3_BUCKET_NAME="bedrock-kb-test-data-${ACCOUNT_ID}"
POLICY_NAME="BedrockKnowledgeBaseTestPolicy"
VECTOR_INDEX_NAME="bedrock-knowledge-base-default-index"

echo -e "${GREEN}🚀 Setting up AWS resources for Knowledge Base testing...${NC}"
echo -e "${BLUE}Account: $ACCOUNT_ID${NC}"
echo -e "${BLUE}Region: $REGION${NC}"
echo -e "${BLUE}Expected setup time: 5-8 minutes${NC}"
echo ""

# Function to check if resource exists and show status
check_resource_exists() {
    local resource_type=$1
    local resource_name=$2
    echo -e "${YELLOW}🔍 Checking if $resource_type '$resource_name' exists...${NC}"
}

# Function to wait with dots
wait_with_dots() {
    local message=$1
    local max_wait=${2:-120}  # Default 2 minutes
    local wait_time=0
    echo -n "$message"
    while [ $wait_time -lt $max_wait ]; do
        echo -n "."
        sleep 2
        wait_time=$((wait_time + 2))
    done
    echo ""
}

# 1. Create IAM Role for Knowledge Base execution
echo -e "${GREEN}Step 1: Creating IAM role for Knowledge Base execution...${NC}"
check_resource_exists "IAM role" "$KB_ROLE_NAME"

if aws iam get-role --role-name "$KB_ROLE_NAME" >/dev/null 2>&1; then
    echo "IAM role $KB_ROLE_NAME already exists"
else
    echo "Creating IAM role $KB_ROLE_NAME..."
    
    # Create trust policy for Bedrock service
    cat > /tmp/kb-trust-policy.json << EOF
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
                    "aws:SourceAccount": "$ACCOUNT_ID"
                }
            }
        }
    ]
}
EOF

    aws iam create-role \
        --role-name "$KB_ROLE_NAME" \
        --assume-role-policy-document file:///tmp/kb-trust-policy.json \
        --description "Execution role for Bedrock Knowledge Base testing"
    
    echo "IAM role created successfully"
fi

# 2. Create and attach IAM policy for Knowledge Base operations
echo -e "${GREEN}Step 2: Creating IAM policy for Knowledge Base operations...${NC}"

cat > /tmp/kb-policy.json << EOF
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
            "Resource": "arn:aws:aoss:$REGION:$ACCOUNT_ID:collection/*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::$S3_BUCKET_NAME",
                "arn:aws:s3:::$S3_BUCKET_NAME/*"
            ]
        }
    ]
}
EOF

# Check if policy exists and create/update it
if aws iam get-role-policy --role-name "$KB_ROLE_NAME" --policy-name "$POLICY_NAME" >/dev/null 2>&1; then
    echo "Updating existing policy..."
    aws iam put-role-policy \
        --role-name "$KB_ROLE_NAME" \
        --policy-name "$POLICY_NAME" \
        --policy-document file:///tmp/kb-policy.json
else
    echo "Creating new policy..."
    aws iam put-role-policy \
        --role-name "$KB_ROLE_NAME" \
        --policy-name "$POLICY_NAME" \
        --policy-document file:///tmp/kb-policy.json
fi

echo "IAM policy configured successfully"

# 3. Create S3 bucket for Knowledge Base data source
echo -e "${GREEN}Step 3: Creating S3 bucket for Knowledge Base data source...${NC}"
check_resource_exists "S3 bucket" "$S3_BUCKET_NAME"

if aws s3 ls "s3://$S3_BUCKET_NAME" >/dev/null 2>&1; then
    echo "S3 bucket $S3_BUCKET_NAME already exists"
else
    echo "Creating S3 bucket $S3_BUCKET_NAME..."
    aws s3 mb "s3://$S3_BUCKET_NAME" --region "$REGION"
    echo "S3 bucket created successfully"
fi

# 4. Upload sample documents to S3
echo -e "${GREEN}Step 4: Uploading sample documents to S3...${NC}"

# Create sample documents
mkdir -p /tmp/kb-docs
cat > /tmp/kb-docs/product-info.txt << EOF
Product Information

Our flagship product, the SuperWidget 3000, is a revolutionary device that combines artificial intelligence with practical utility. 

Features:
- Advanced AI processing capabilities
- 24/7 customer support integration
- Cloud-based data synchronization
- Mobile app compatibility
- Energy-efficient design

The SuperWidget 3000 is available in three configurations:
1. Basic Edition: $299 - Perfect for individual users
2. Pro Edition: $599 - Ideal for small businesses  
3. Enterprise Edition: $1299 - Full-featured for large organizations

Installation is simple and typically takes 15-30 minutes. Our certified technicians are available for on-site installation if needed.

For technical support, customers can contact our 24/7 helpdesk at support@example.com or call 1-800-WIDGET.
EOF

cat > /tmp/kb-docs/faq.txt << EOF
Frequently Asked Questions

Q: How do I set up my SuperWidget 3000?
A: Setup is straightforward. Simply connect the device to power, download our mobile app, and follow the guided setup process. The entire process typically takes 15-30 minutes.

Q: What warranty is included?
A: All SuperWidget 3000 devices come with a 2-year manufacturer warranty covering hardware defects and software issues.

Q: Can I use SuperWidget 3000 with my existing systems?
A: Yes! SuperWidget 3000 is designed to integrate seamlessly with most existing systems. Our compatibility checker in the mobile app can verify compatibility with your specific setup.

Q: How do I contact customer support?
A: Our customer support team is available 24/7 via:
- Email: support@example.com
- Phone: 1-800-WIDGET
- Live chat through our mobile app
- Online support portal at support.example.com

Q: What's the difference between the editions?
A: Basic Edition includes core functionality, Pro Edition adds business features like team management and advanced analytics, while Enterprise Edition includes all features plus priority support and custom integrations.

Q: Is my data secure?
A: Absolutely. We use enterprise-grade encryption for all data transmission and storage. All data is processed in compliance with industry security standards.
EOF

# Upload documents to S3
echo "Uploading sample documents..."
aws s3 cp /tmp/kb-docs/ "s3://$S3_BUCKET_NAME/docs/" --recursive
echo "Sample documents uploaded successfully"

# 5. Create OpenSearch Serverless security policies
echo -e "${GREEN}Step 5: Creating OpenSearch Serverless security policies...${NC}"

# Encryption policy
ENCRYPTION_POLICY_NAME="kb-collection-encryption-policy"
cat > /tmp/encryption-policy.json << EOF
{
    "Rules": [
        {
            "Resource": ["collection/$COLLECTION_NAME"],
            "ResourceType": "collection"
        }
    ],
    "AWSOwnedKey": true
}
EOF

if aws opensearchserverless get-security-policy --name "$ENCRYPTION_POLICY_NAME" --type encryption >/dev/null 2>&1; then
    echo "Encryption policy already exists"
else
    echo "Creating encryption policy..."
    aws opensearchserverless create-security-policy \
        --name "$ENCRYPTION_POLICY_NAME" \
        --type encryption \
        --policy file:///tmp/encryption-policy.json \
        --description "Encryption policy for KB testing collection"
fi

# Network policy
NETWORK_POLICY_NAME="kb-collection-network-policy"
cat > /tmp/network-policy.json << EOF
[
    {
        "Rules": [
            {
                "Resource": ["collection/$COLLECTION_NAME"],
                "ResourceType": "collection"
            }
        ],
        "AllowFromPublic": true
    }
]
EOF

if aws opensearchserverless get-security-policy --name "$NETWORK_POLICY_NAME" --type network >/dev/null 2>&1; then
    echo "Network policy already exists"
else
    echo "Creating network policy..."
    aws opensearchserverless create-security-policy \
        --name "$NETWORK_POLICY_NAME" \
        --type network \
        --policy file:///tmp/network-policy.json \
        --description "Network policy for KB testing collection"
fi

# 6. Create OpenSearch Serverless collection
echo -e "${GREEN}Step 6: Creating OpenSearch Serverless collection...${NC}"
check_resource_exists "OpenSearch Serverless collection" "$COLLECTION_NAME"

# Check if collection exists and get its details
EXISTING_COLLECTION=$(aws opensearchserverless list-collections --collection-filters name="$COLLECTION_NAME" --query 'collectionSummaries[0]' --output json 2>/dev/null || echo "null")

if [ "$EXISTING_COLLECTION" != "null" ] && [ "$EXISTING_COLLECTION" != "" ]; then
    echo "✅ Collection $COLLECTION_NAME already exists"
    COLLECTION_ID=$(echo "$EXISTING_COLLECTION" | jq -r '.id // empty')
    COLLECTION_STATUS=$(echo "$EXISTING_COLLECTION" | jq -r '.status // empty')
    
    if [ "$COLLECTION_STATUS" != "ACTIVE" ]; then
        echo "⏳ Collection exists but is not ACTIVE (status: $COLLECTION_STATUS), waiting..."
        
        # Wait for existing collection to become active
        WAIT_TIME=0
        MAX_WAIT=300  # 5 minutes
        
        while [ $WAIT_TIME -lt $MAX_WAIT ]; do
            STATUS=$(aws opensearchserverless list-collections --collection-filters name="$COLLECTION_NAME" --query 'collectionSummaries[0].status' --output text 2>/dev/null || echo "UNKNOWN")
            
            case "$STATUS" in
                "ACTIVE")
                    echo ""
                    echo "✅ Collection is now ACTIVE"
                    break
                    ;;
                "FAILED")
                    echo ""
                    echo -e "${RED}❌ Collection creation failed${NC}"
                    exit 1
                    ;;
                *)
                    echo -n "."
                    sleep 10
                    WAIT_TIME=$((WAIT_TIME + 10))
                    ;;
            esac
        done
        
        if [ $WAIT_TIME -ge $MAX_WAIT ]; then
            echo ""
            echo -e "${RED}❌ Timeout waiting for collection to become ACTIVE${NC}"
            exit 1
        fi
    fi
else
    echo "📝 Creating collection $COLLECTION_NAME..."
    COLLECTION_RESPONSE=$(aws opensearchserverless create-collection \
        --name "$COLLECTION_NAME" \
        --type VECTORSEARCH \
        --description "Vector collection for Knowledge Base testing")
    
    COLLECTION_ID=$(echo "$COLLECTION_RESPONSE" | jq -r '.createCollectionDetail.id')
    
    echo "✅ Collection created with ID: $COLLECTION_ID"
    
    # Wait for collection to become active with better feedback
    echo "⏳ Waiting for collection to become ACTIVE..."
    WAIT_TIME=0
    MAX_WAIT=300  # 5 minutes
    
    while [ $WAIT_TIME -lt $MAX_WAIT ]; do
        STATUS=$(aws opensearchserverless list-collections --collection-filters name="$COLLECTION_NAME" --query 'collectionSummaries[0].status' --output text 2>/dev/null || echo "UNKNOWN")
        
        case "$STATUS" in
            "ACTIVE")
                echo ""
                echo "✅ Collection is now ACTIVE"
                break
                ;;
            "FAILED")
                echo ""
                echo -e "${RED}❌ Collection creation failed${NC}"
                exit 1
                ;;
            "CREATING")
                echo -n "."
                sleep 10
                WAIT_TIME=$((WAIT_TIME + 10))
                ;;
            *)
                echo -n "?"
                sleep 5
                WAIT_TIME=$((WAIT_TIME + 5))
                ;;
        esac
    done
    
    if [ $WAIT_TIME -ge $MAX_WAIT ]; then
        echo ""
        echo -e "${RED}❌ Timeout waiting for collection to become ACTIVE${NC}"
        exit 1
    fi
fi

# Set collection details for later use
COLLECTION_ID=${COLLECTION_ID:-$(aws opensearchserverless list-collections --collection-filters name="$COLLECTION_NAME" --query 'collectionSummaries[0].id' --output text)}
COLLECTION_ARN="arn:aws:aoss:$REGION:$ACCOUNT_ID:collection/$COLLECTION_ID"
COLLECTION_ENDPOINT="$COLLECTION_ID.$REGION.aoss.amazonaws.com"

echo -e "${BLUE}📍 Collection ARN: $COLLECTION_ARN${NC}"
echo -e "${BLUE}🌐 Collection Endpoint: $COLLECTION_ENDPOINT${NC}"

# 7. Create data access policy for the collection
echo -e "${GREEN}Step 7: Creating data access policy for OpenSearch Serverless...${NC}"

POLICY_NAME_OSS="kb-collection-access-policy"

cat > /tmp/data-access-policy.json << EOF
[
    {
        "Rules": [
            {
                "Resource": ["collection/$COLLECTION_NAME"],
                "Permission": [
                    "aoss:CreateCollectionItems",
                    "aoss:DeleteCollectionItems", 
                    "aoss:UpdateCollectionItems",
                    "aoss:DescribeCollectionItems"
                ],
                "ResourceType": "collection"
            },
            {
                "Resource": ["index/$COLLECTION_NAME/*"],
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
            "arn:aws:iam::$ACCOUNT_ID:role/$KB_ROLE_NAME"
        ],
        "Description": "Data access policy for Knowledge Base testing"
    }
]
EOF

# Check if data access policy exists
if aws opensearchserverless get-access-policy --name "$POLICY_NAME_OSS" --type data >/dev/null 2>&1; then
    echo "Updating existing data access policy..."
    aws opensearchserverless update-access-policy \
        --name "$POLICY_NAME_OSS" \
        --type data \
        --policy file:///tmp/data-access-policy.json
else
    echo "Creating data access policy..."
    aws opensearchserverless create-access-policy \
        --name "$POLICY_NAME_OSS" \
        --type data \
        --policy file:///tmp/data-access-policy.json \
        --description "Data access policy for KB testing"
fi

echo "Data access policy configured successfully"

# 8. Output configuration for Knowledge Base CR
echo -e "${GREEN}Step 8: Generating Knowledge Base configuration...${NC}"

KB_ROLE_ARN="arn:aws:iam::$ACCOUNT_ID:role/$KB_ROLE_NAME"

cat > kb-test-config.yaml << EOF
# AWS Resources created for Knowledge Base testing
# Generated on: $(date)
# Account: $ACCOUNT_ID | Region: $REGION
apiVersion: bedrock.aws.example.com/v1
kind: BedrockResource
metadata:
  name: test-knowledge-base
  namespace: default
spec:
  type: knowledgeBase
  knowledgeBase:
    name: "TestKnowledgeBase"
    description: "Knowledge base for integration testing with automated OpenSearch index creation"
    roleArn: "$KB_ROLE_ARN"
    embeddingModelArn: "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v1"
    vectorStoreType: "OPENSEARCH_SERVERLESS"
    opensearchServerlessConfiguration:
      collectionArn: "$COLLECTION_ARN"
      vectorIndexName: "$VECTOR_INDEX_NAME"
      vectorField: "vector"
      textField: "text"
      metadataField: "metadata"
    tags:
      Environment: "test"
      Purpose: "integration-testing"
      CreatedBy: "bedrock-controller"
      TestSession: "$(date +%Y%m%d-%H%M%S)"
---
# Configuration summary
# Account ID: $ACCOUNT_ID
# Region: $REGION  
# KB Role ARN: $KB_ROLE_ARN
# Collection ARN: $COLLECTION_ARN
# Collection Endpoint: $COLLECTION_ENDPOINT
# Vector Index Name: $VECTOR_INDEX_NAME
# S3 Bucket: $S3_BUCKET_NAME
EOF

echo -e "${GREEN}🎉 All AWS resources for Knowledge Base testing have been set up successfully!${NC}"
echo ""
echo -e "${BLUE}📋 Resources created:${NC}"
echo -e "  🔐 IAM Role: ${YELLOW}$KB_ROLE_ARN${NC}"
echo -e "  🪣 S3 Bucket: ${YELLOW}$S3_BUCKET_NAME${NC}"
echo -e "  🔍 OpenSearch Collection: ${YELLOW}$COLLECTION_ARN${NC}"
echo -e "  🌐 Collection Endpoint: ${YELLOW}$COLLECTION_ENDPOINT${NC}"
echo -e "  📑 Vector Index Name: ${YELLOW}$VECTOR_INDEX_NAME${NC}"
echo -e "  📄 Sample documents uploaded to S3"
echo -e "  ⚙️  Knowledge Base config: ${YELLOW}kb-test-config.yaml${NC}"
echo ""
echo "Knowledge Base test configuration saved to: kb-test-config.yaml"
echo ""
echo "You can now run Knowledge Base integration tests using the generated configuration."

# Cleanup temp files
rm -f /tmp/kb-trust-policy.json /tmp/kb-policy.json /tmp/data-access-policy.json
rm -f /tmp/encryption-policy.json /tmp/network-policy.json
rm -rf /tmp/kb-docs

echo -e "${GREEN}Setup complete!${NC}"
