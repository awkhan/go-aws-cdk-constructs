package website

import (
	"fmt"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfrontorigins"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53targets"
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
	HostedZone  awsroute53.IHostedZone
}

type Website struct {
	constructs.Construct
}

func New(scope constructs.Construct, id string, options Options) Website {

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
		FunctionName: jsii.String(fmt.Sprintf("cdkUrlRewriter-%s", id)),
	})
	functionAssoc := awscloudfront.FunctionAssociation{
		EventType: awscloudfront.FunctionEventType_VIEWER_REQUEST,
		Function:  function,
	}

	distribution := awscloudfront.NewDistribution(this, jsii.String("cloudfront-distribution"), &awscloudfront.DistributionProps{
		DefaultBehavior: &awscloudfront.BehaviorOptions{
			FunctionAssociations: &[]*awscloudfront.FunctionAssociation{&functionAssoc}, //slightly awk
			ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
			Origin:               bucketOrigin,
		},
		Certificate: options.Certificate,
		DomainNames: jsii.Strings(options.DomainName, fmt.Sprintf("www.%s", options.DomainName)),
	})

	cfTarget := awsroute53targets.NewCloudFrontTarget(distribution)
	awsroute53.NewARecord(this, jsii.String("distribution-a-record"), &awsroute53.ARecordProps{
		Zone:   options.HostedZone,
		Ttl:    awscdk.Duration_Seconds(jsii.Number(60)),
		Target: awsroute53.RecordTarget_FromAlias(cfTarget),
	})

	awsroute53.NewCnameRecord(this, jsii.String("www-redirect"), &awsroute53.CnameRecordProps{
		Zone:       options.HostedZone,
		Ttl:        awscdk.Duration_Seconds(jsii.Number(60)),
		RecordName: jsii.String("www"),
		DomainName: jsii.String(options.DomainName),
	})

	return Website{this}

}
