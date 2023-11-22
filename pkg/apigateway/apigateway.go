package apigateway

import (
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
	Certificate awscertificatemanager.ICertificate
	HostedZone  awsroute53.IHostedZone
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

	api := awsapigateway.NewRestApi(this, jsii.String("rest-api"), &awsapigateway.RestApiProps{
		DeployOptions: &awsapigateway.StageOptions{
			MetricsEnabled:   jsii.Bool(true),
			LoggingLevel:     awsapigateway.MethodLoggingLevel_INFO,
			DataTraceEnabled: jsii.Bool(true),
		},
		CloudWatchRole: jsii.Bool(true),
		DomainName: &awsapigateway.DomainNameOptions{
			Certificate:  options.Certificate,
			DomainName:   options.HostedZone.ZoneName(),
			EndpointType: "EDGE",
		},
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
