#!/bin/bash

# Create vector index in OpenSearch Serverless using AWS CLI and curl
set -e

# Configuration
COLLECTION_ENDPOINT="https://REPLACE_ME_COLLECTION_ID.us-east-1.aoss.amazonaws.com"
INDEX_NAME="test-vector-index"
REGION="us-east-1"

echo "Creating vector index '$INDEX_NAME' in OpenSearch Serverless collection..."

# Create index mapping for Knowledge Base
cat > /tmp/index-mapping.json << 'EOF'
{
    "settings": {
        "index": {
            "knn": true,
            "knn.algo_param.ef_search": 256
        }
    },
    "mappings": {
        "properties": {
            "vector": {
                "type": "knn_vector",
                "dimension": 1536,
                "method": {
                    "name": "hnsw",
                    "space_type": "cosinesimil",
                    "engine": "nmslib",
                    "parameters": {
                        "ef_construction": 256,
                        "m": 16
                    }
                }
            },
            "text": {
                "type": "text"
            },
            "metadata": {
                "type": "object"
            }
        }
    }
}
EOF

# Check if index exists first
echo "Checking if index '$INDEX_NAME' already exists..."
STATUS_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    --aws-sigv4 "aws:amz:${REGION}:aoss" \
    --user "$(aws configure get aws_access_key_id):$(aws configure get aws_secret_access_key)" \
    -H "Content-Type: application/json" \
    "$COLLECTION_ENDPOINT/$INDEX_NAME")

if [ "$STATUS_CODE" = "200" ]; then
    echo "Index '$INDEX_NAME' already exists"
    echo "✅ Vector index is ready!"
    rm -f /tmp/index-mapping.json
    exit 0
elif [ "$STATUS_CODE" = "404" ]; then
    echo "Index '$INDEX_NAME' does not exist, creating..."
elif [ "$STATUS_CODE" = "403" ]; then
    echo "❌ Access denied to OpenSearch collection. Check IAM permissions and data access policies."
    rm -f /tmp/index-mapping.json
    exit 1
else
    echo "Unexpected status code when checking index: $STATUS_CODE"
fi

# Create the index
echo "Creating index with vector mapping..."
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    --aws-sigv4 "aws:amz:${REGION}:aoss" \
    --user "$(aws configure get aws_access_key_id):$(aws configure get aws_secret_access_key)" \
    -X PUT \
    -H "Content-Type: application/json" \
    -d @/tmp/index-mapping.json \
    "$COLLECTION_ENDPOINT/$INDEX_NAME")

HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS:" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS:/d')

if [ "$HTTP_STATUS" = "200" ] || [ "$HTTP_STATUS" = "201" ]; then
    echo "✅ Successfully created index '$INDEX_NAME'"
    echo "Response: $BODY"
else
    echo "❌ Failed to create index: HTTP $HTTP_STATUS"
    echo "Response: $BODY"
    rm -f /tmp/index-mapping.json
    exit 1
fi

# Cleanup
rm -f /tmp/index-mapping.json

echo "✅ Vector index setup completed successfully!"
