package apigateway

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type Options struct {
	awscdk.StackProps
}

type APIGateway struct {
	constructs.Construct
	API awsapigateway.RestApi
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
	})

	return APIGateway{this, api}

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
	resource.AddMethod(jsii.String(method), integration, &awsapigateway.MethodOptions{
		AuthorizationType: "CUSTOM",
		Authorizer:        authorizer,
	})
}
