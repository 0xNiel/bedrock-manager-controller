module github.com/odnielgonzalez/bedrock-manager-controller

go 1.22

require (
	github.com/aws/aws-sdk-go-v2 v1.24.0
	github.com/aws/aws-sdk-go-v2/config v1.26.1
	github.com/aws/aws-sdk-go-v2/service/bedrock v1.7.2
	github.com/aws/aws-sdk-go-v2/service/bedrockagent v1.7.3
	k8s.io/api v0.29.0
	k8s.io/apimachinery v0.29.0
	k8s.io/client-go v0.29.0
	k8s.io/klog/v2 v2.110.1
	sigs.k8s.io/controller-runtime v0.16.3
)

require (
	github.com/aws/aws-sdk-go-v2/credentials v1.16.12
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.10
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.9
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.9
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.2
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.9
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.5
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.5
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.5
	github.com/aws/smithy-go v1.19.0
)
