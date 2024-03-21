package apigateway

import (
	"fmt"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53targets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type Options struct {
	awscdk.StackProps
	APIName     string
	Certificate awscertificatemanager.ICertificate
	HostedZone  awsroute53.IHostedZone
	Authorizer  awslambda.IFunction
}

type APIGateway struct {
	constructs.Construct
	API awsapigateway.RestApi
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
	if options.Authorizer != nil {
		authorizer := awsapigateway.NewRequestAuthorizer(this, jsii.String("request-authorizer"), &awsapigateway.RequestAuthorizerProps{
			Handler:         options.Authorizer,
			AuthorizerName:  jsii.String(fmt.Sprintf("%s-authorizer", options.APIName)),
			ResultsCacheTtl: awscdk.Duration_Seconds(jsii.Number(30)),
			IdentitySources: &[]*string{awsapigateway.IdentitySource_Header(jsii.String("authorizer"))},
		})
		methodOptions = &awsapigateway.MethodOptions{Authorizer: authorizer}
	}

	api := awsapigateway.NewRestApi(this, jsii.String(options.APIName), &awsapigateway.RestApiProps{
		DeployOptions: &awsapigateway.StageOptions{
			MetricsEnabled:   jsii.Bool(true),
			LoggingLevel:     awsapigateway.MethodLoggingLevel_INFO,
			DataTraceEnabled: jsii.Bool(true),
		},
		CloudWatchRole: jsii.Bool(true),
		DomainName: &awsapigateway.DomainNameOptions{
			Certificate:  options.Certificate,
			DomainName:   jsii.String(fmt.Sprintf("api.%s", *options.HostedZone.ZoneName())),
			EndpointType: "EDGE",
		},
		DefaultMethodOptions: methodOptions,
	})

	awsroute53.NewARecord(this, jsii.String("route53-a-record"), &awsroute53.ARecordProps{
		Zone:           options.HostedZone,
		Comment:        nil,
		DeleteExisting: nil,
		GeoLocation:    nil,
		RecordName:     jsii.String("api"),
		Ttl:            nil,
		Target:         awsroute53.RecordTarget_FromAlias(awsroute53targets.NewApiGateway(api)),
	})

	return APIGateway{this, api}

}

func (a *APIGateway) AddLambdaIntegrations(integrations []LambdaIntegration) {
	for _, v := range integrations {
		a.AddLambdaIntegration(v.Function, v.Path, v.Method, v.Authorizer)
	}
}

func (a *APIGateway) AddLambdaIntegration(handler awslambda.IFunction, path, method string, authorizer awsapigateway.IAuthorizer) {
	integration := awsapigateway.NewLambdaIntegration(handler, &awsapigateway.LambdaIntegrationOptions{})
	resource := a.API.Root().AddResource(jsii.String(path), &awsapigateway.ResourceOptions{
		DefaultCorsPreflightOptions: &awsapigateway.CorsOptions{
			AllowOrigins:     jsii.Strings("*"),
			AllowCredentials: jsii.Bool(true),
			AllowHeaders:     jsii.Strings("*"),
			AllowMethods:     jsii.Strings("*"),
			StatusCode:       jsii.Number(201),
		},
		DefaultIntegration:   nil,
		DefaultMethodOptions: nil,
	})
	options := &awsapigateway.MethodOptions{}
	if authorizer != nil {
		options.AuthorizationType = "CUSTOM"
		options.Authorizer = authorizer
	}
	resource.AddMethod(jsii.String(method), integration, options)
}

func AddLambdaIntegrationsToAPIGateway(api awsapigateway.IRestApi, integrations []LambdaIntegration) {
	for _, v := range integrations {
		AddLambdaIntegrationToAPIGateway(api, v.Function, v.Path, v.Method, v.Authorizer)
	}
}

func AddLambdaIntegrationToAPIGateway(api awsapigateway.IRestApi, handler awslambda.IFunction, path, method string, authorizer awsapigateway.IAuthorizer) {
	integration := awsapigateway.NewLambdaIntegration(handler, &awsapigateway.LambdaIntegrationOptions{})
	resource := api.Root().AddResource(jsii.String(path), &awsapigateway.ResourceOptions{
		DefaultCorsPreflightOptions: &awsapigateway.CorsOptions{
			AllowOrigins:     jsii.Strings("*"),
			AllowCredentials: jsii.Bool(true),
			AllowHeaders:     jsii.Strings("*"),
			AllowMethods:     jsii.Strings("*"),
			StatusCode:       jsii.Number(201),
		},
		DefaultIntegration:   nil,
		DefaultMethodOptions: nil,
	})
	options := &awsapigateway.MethodOptions{}
	if authorizer != nil {
		options.AuthorizationType = "CUSTOM"
		options.Authorizer = authorizer
	}
	resource.AddMethod(jsii.String(method), integration, options)
}
