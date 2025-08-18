#!/usr/bin/env python3

import json
import requests
import boto3
from botocore.auth import SigV4Auth
from botocore.awsrequest import AWSRequest
import sys

# Configuration
COLLECTION_ENDPOINT = "https://REPLACE_ME_COLLECTION_ID.us-east-1.aoss.amazonaws.com"
INDEX_NAME = "test-vector-index"
REGION = "us-east-1"

# Create index mapping for Knowledge Base
index_mapping = {
    "settings": {
        "index": {
            "knn": True,
            "knn.algo_param.ef_search": 256
        }
    },
    "mappings": {
        "properties": {
            "vector": {
                "type": "knn_vector",
                "dimension": 1536,  # Amazon Titan embedding dimension
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

def create_signed_request(method, url, body=None):
    """Create a signed request for OpenSearch Serverless."""
    session = boto3.Session()
    credentials = session.get_credentials()
    
    request = AWSRequest(method=method, url=url, data=body)
    SigV4Auth(credentials, "aoss", REGION).add_auth(request)
    
    return request

def create_index():
    """Create the vector index in OpenSearch Serverless."""
    url = f"{COLLECTION_ENDPOINT}/{INDEX_NAME}"
    
    print(f"Creating index '{INDEX_NAME}' at {url}")
    
    # Check if index already exists
    check_request = create_signed_request("HEAD", url)
    prepared_request = check_request.prepare()
    
    response = requests.request(
        prepared_request.method,
        prepared_request.url,
        headers=prepared_request.headers
    )
    
    if response.status_code == 200:
        print(f"Index '{INDEX_NAME}' already exists")
        return True
    elif response.status_code == 404:
        print(f"Index '{INDEX_NAME}' does not exist, creating...")
    else:
        print(f"Error checking index existence: {response.status_code} - {response.text}")
        return False
    
    # Create the index
    body = json.dumps(index_mapping)
    create_request = create_signed_request("PUT", url, body)
    prepared_request = create_request.prepare()
    
    response = requests.request(
        prepared_request.method,
        prepared_request.url,
        headers=prepared_request.headers,
        data=prepared_request.body
    )
    
    if response.status_code in [200, 201]:
        print(f"Successfully created index '{INDEX_NAME}'")
        print("Index response:", response.text)
        return True
    else:
        print(f"Failed to create index: {response.status_code} - {response.text}")
        return False

if __name__ == "__main__":
    print("Setting up vector index for Knowledge Base testing...")
    
    try:
        success = create_index()
        if success:
            print("✅ Vector index setup completed successfully!")
            sys.exit(0)
        else:
            print("❌ Vector index setup failed!")
            sys.exit(1)
    except Exception as e:
        print(f"❌ Error: {e}")
        sys.exit(1)
