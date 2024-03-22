package apigateway

import (
	"fmt"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53targets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"time"
)

type Options struct {
	awscdk.StackProps
	APIName    string
	Authorizer awslambda.IFunction
}

type APIGateway struct {
	constructs.Construct
	API        awsapigateway.RestApi
	Authorizer awsapigateway.IAuthorizer
}

type LambdaIntegration struct {
	Function   awslambda.IFunction
	Path       string
	Method     string
	Authorizer awsapigateway.IAuthorizer
}

func New(scope constructs.Construct, id string, options Options) APIGateway {

	this := constructs.NewConstruct(scope, &id)

	var methodOptions *awsapigateway.MethodOptions
	var authorizer awsapigateway.IAuthorizer
	if options.Authorizer != nil {
		authorizer = awsapigateway.NewRequestAuthorizer(this, jsii.String("request-authorizer"), &awsapigateway.RequestAuthorizerProps{
			Handler:         options.Authorizer,
			AuthorizerName:  jsii.String(fmt.Sprintf("%s-authorizer", options.APIName)),
			ResultsCacheTtl: awscdk.Duration_Seconds(jsii.Number(30)),
			IdentitySources: &[]*string{awsapigateway.IdentitySource_Header(jsii.String("authorization"))},
		})
		methodOptions = &awsapigateway.MethodOptions{Authorizer: authorizer}
	}

	api := awsapigateway.NewRestApi(this, jsii.String(options.APIName), &awsapigateway.RestApiProps{
		CloudWatchRole:       jsii.Bool(true),
		DefaultMethodOptions: methodOptions,
		Deploy:               jsii.Bool(false),
	})

	if options.Authorizer != nil {
		options.Authorizer.AddPermission(jsii.String("api-gateway-invoke"), &awslambda.Permission{
			Principal: awsiam.NewServicePrincipal(jsii.String("apigateway.amazonaws.com"), nil),
			SourceArn: api.ArnForExecuteApi(jsii.String("*"), jsii.String("/*"), jsii.String("*")),
		})
	}

	api.Root().AddMethod(jsii.String("ANY"), nil, nil)

	return APIGateway{Construct: this, API: api, Authorizer: authorizer}

}

type DeploymentOptions struct {
	*awscdk.StackProps
	Certificate  awscertificatemanager.ICertificate
	HostedZone   awsroute53.IHostedZone
	RestAPI      awsapigateway.IRestApi
	Integrations []LambdaIntegration
}

type Deployment struct {
	constructs.Construct
}

func NewDeployment(scope constructs.Construct, id string, options DeploymentOptions) Deployment {

	this := constructs.NewConstruct(scope, &id)

	api := awsapigateway.RestApi_FromRestApiAttributes(this, jsii.String("rest-api"), &awsapigateway.RestApiAttributes{
		RestApiId:      options.RestAPI.RestApiId(),
		RootResourceId: options.RestAPI.Root().ResourceId(),
	})

	for _, v := range options.Integrations {
		AddLambdaIntegrationToAPIGateway(api, v.Function, v.Path, v.Method, v.Authorizer)
	}

	deployment := awsapigateway.NewDeployment(this, jsii.String(fmt.Sprintf("api-gw-deployment-%s", time.Now().String())), &awsapigateway.DeploymentProps{
		Api:         api,
		Description: jsii.String("Deployment"),
	})

	stage := awsapigateway.NewStage(this, jsii.String("api-gw-stage"), &awsapigateway.StageProps{
		DataTraceEnabled: jsii.Bool(true),
		LoggingLevel:     awsapigateway.MethodLoggingLevel_INFO,
		MetricsEnabled:   jsii.Bool(true),
		StageName:        jsii.String("prod"),
		Deployment:       deployment,
	})

	api.SetDeploymentStage(stage)

	domainName := awsapigateway.NewDomainName(this, jsii.String("domain-name"), &awsapigateway.DomainNameProps{
		Certificate:  options.Certificate,
		DomainName:   jsii.String(fmt.Sprintf("api.%s", *options.HostedZone.ZoneName())),
		EndpointType: "EDGE",
	})
	domainName.AddBasePathMapping(api, &awsapigateway.BasePathMappingOptions{
		AttachToStage: jsii.Bool(true),
		Stage:         stage,
	})

	awsroute53.NewARecord(this, jsii.String("route53-a-record"), &awsroute53.ARecordProps{
		Zone:       options.HostedZone,
		RecordName: jsii.String("api"),
		Target:     awsroute53.RecordTarget_FromAlias(awsroute53targets.NewApiGatewayDomain(domainName)),
	})

	return Deployment{this}

}

func AddLambdaIntegrationToAPIGateway(api awsapigateway.IRestApi, handler awslambda.IFunction, path, method string, authorizer awsapigateway.IAuthorizer) {

	integration := awsapigateway.NewLambdaIntegration(handler, &awsapigateway.LambdaIntegrationOptions{})

	resource := api.Root().ResourceForPath(jsii.String(path))
	resource.AddCorsPreflight(&awsapigateway.CorsOptions{
		AllowOrigins:     jsii.Strings("*"),
		AllowCredentials: jsii.Bool(true),
		AllowHeaders:     jsii.Strings("*"),
		AllowMethods:     jsii.Strings("*"),
		StatusCode:       jsii.Number(201),
	})

	options := &awsapigateway.MethodOptions{}
	if authorizer != nil {
		options.AuthorizationType = "CUSTOM"
		options.Authorizer = authorizer
	}

	resource.AddMethod(jsii.String(method), integration, options)

}
