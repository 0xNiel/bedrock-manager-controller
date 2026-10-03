# Deployment Guide

This directory contains Kubernetes deployment manifests for the Unified AWS Bedrock Controller.

## Prerequisites

1. **Kubernetes cluster** (1.19+)
2. **AWS IAM role** with Bedrock permissions
3. **IRSA (IAM Roles for Service Accounts)** configured on EKS, or alternative credential method

## IAM Role Configuration

### Option 1: IRSA (Recommended for EKS)

1. **Create Controller IAM Role**: For the controller to call AWS APIs
2. **Create Agent Execution Role**: For Bedrock agents to assume

2. **Attach IAM Policy** (ensure the role has these permissions):
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "bedrock:*",
                "iam:PassRole"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": "arn:aws:logs:*:*:*"
        }
    ]
}
```

3. **Configure IRSA Trust Relationship**:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Federated": "arn:aws:iam::REPLACE_ME_AWS_ACCOUNT_ID:oidc-provider/oidc.eks.REGION.amazonaws.com/id/CLUSTER_ID"
            },
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Condition": {
                "StringEquals": {
                    "oidc.eks.REGION.amazonaws.com/id/CLUSTER_ID:sub": "system:serviceaccount:bedrock-system:bedrock-controller",
                    "oidc.eks.REGION.amazonaws.com/id/CLUSTER_ID:aud": "sts.amazonaws.com"
                }
            }
        }
    ]
}
```

### Option 2: Environment Variables

Set AWS credentials as environment variables in the deployment:
```yaml
env:
- name: AWS_ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      name: aws-credentials
      key: access-key-id
- name: AWS_SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      name: aws-credentials
      key: secret-access-key
- name: AWS_REGION
  value: "us-east-1"
```

### Option 3: Instance Profile (EC2 nodes)

If running on EC2 nodes, attach the IAM role to the instance profile.

## Deployment Steps

0. **Build and push the image**, then set it in `deployment.yaml` (marked `REPLACE ME`):
```bash
make -C .. docker-push IMAGE_REGISTRY=<your-registry>
```

1. **Create namespace**:
```bash
kubectl create namespace bedrock-system
```

2. **Apply CRD**:
```bash
kubectl apply -f ../crd/bedrockresource.yaml
```

3. **Create service account** (for IRSA):
```bash
kubectl apply -f serviceaccount.yaml
```

4. **Apply RBAC**:
```bash
kubectl apply -f rbac.yaml
```

5. **Deploy controller**:
```bash
kubectl apply -f deployment.yaml
```

6. **Verify deployment**:
```bash
kubectl get pods -n bedrock-system
kubectl logs -f deployment/bedrock-controller -n bedrock-system
```

## Testing

1. **Apply a test resource**:
```bash
kubectl apply -f ../cr/bedrockagent-sample.yaml
```

2. **Check resource status**:
```bash
kubectl get bedrockresources
kubectl describe bedrockresource customer-service-agent
```

3. **Verify in AWS Console**:
Check the Bedrock console to confirm the agent was created.

## Troubleshooting

### Common Issues

1. **Permission Denied**:
   - Verify IAM role has correct permissions
   - Check IRSA trust relationship
   - Ensure service account annotation is correct

2. **Region Mismatch**:
   - Verify AWS_REGION environment variable
   - Check if Bedrock is available in the target region

3. **Controller Not Starting**:
   - Check pod logs: `kubectl logs deployment/bedrock-controller -n bedrock-system`
   - Verify CRD was applied successfully
   - Check RBAC permissions

### Logs and Debugging

```bash
# View controller logs
kubectl logs -f deployment/bedrock-controller -n bedrock-system

# Check events
kubectl get events -n bedrock-system

# Describe problematic resources
kubectl describe bedrockresource <resource-name>
```
