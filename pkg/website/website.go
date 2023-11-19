package website

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfrontorigins"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3deployment"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type Options struct {
	awscdk.StackProps
	DomainName  string
	AssetPath   string
	Certificate awscertificatemanager.ICertificate
}

type Website struct {
	constructs.Construct
}

func New(scope constructs.Construct, id string, options *Options) Website {

	this := constructs.NewConstruct(scope, &id)

	bucket := awss3.NewBucket(this, jsii.String("bucket"), &awss3.BucketProps{
		AccessControl: awss3.BucketAccessControl_PRIVATE,
	})

	sourceAsset := awss3deployment.Source_Asset(&options.AssetPath, &awss3assets.AssetOptions{})
	awss3deployment.NewBucketDeployment(this, jsii.String("bucket-deployment"), &awss3deployment.BucketDeploymentProps{
		DestinationBucket: bucket,
		Sources:           &[]awss3deployment.ISource{sourceAsset},
	})

	cfOAI := awscloudfront.NewOriginAccessIdentity(this, jsii.String("cloudfront-origin-access-identity"), &awscloudfront.OriginAccessIdentityProps{})

	policyStatement := awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:    jsii.Strings("s3:GetObject"),
		Conditions: &map[string]interface{}{},
		Effect:     awsiam.Effect_ALLOW,
		Principals: &[]awsiam.IPrincipal{cfOAI.GrantPrincipal()},
		Resources:  &[]*string{bucket.BucketArn()},
	})
	bucket.AddToResourcePolicy(policyStatement)

	bucketOrigin := awscloudfrontorigins.NewS3Origin(bucket, &awscloudfrontorigins.S3OriginProps{
		ConnectionAttempts:   jsii.Number(2),
		OriginAccessIdentity: cfOAI,
	})

	function := awscloudfront.NewFunction(this, jsii.String("cloudfront-function"), &awscloudfront.FunctionProps{
		Code:         awscloudfront.FunctionCode_FromInline(jsii.String("function handler(event) {var request = event.request; var uri = request.uri; if (uri.endsWith('/')) {request.uri += 'index.html';} else if (!uri.includes('.')) {request.uri += '/index.html';};return request;}")),
		Comment:      jsii.String("Rewrite the uri to add index.html after viewer request"),
		FunctionName: jsii.String("cdkUrlRewriter"),
	})
	functionAssoc := awscloudfront.FunctionAssociation{
		EventType: awscloudfront.FunctionEventType_VIEWER_REQUEST,
		Function:  function,
	}

	awscloudfront.NewDistribution(this, jsii.String("cloudfront-distribution"), &awscloudfront.DistributionProps{
		DefaultBehavior: &awscloudfront.BehaviorOptions{
			FunctionAssociations: &[]*awscloudfront.FunctionAssociation{&functionAssoc}, //slightly awk
			ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
			Origin:               bucketOrigin,
		},
		AdditionalBehaviors: &map[string]*awscloudfront.BehaviorOptions{},
		Certificate:         options.Certificate,
		Comment:             new(string),
		DefaultRootObject:   new(string),
		DomainNames:         jsii.Strings(options.DomainName),
	})

	return Website{this}

}
