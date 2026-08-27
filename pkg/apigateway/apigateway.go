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
	"time"
)

type Options struct {
	*awscdk.StackProps
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

	awsapigateway.NewGatewayResponse(this, jsii.String("access-denied-gateway-response"), &awsapigateway.GatewayResponseProps{
		Type: awsapigateway.ResponseType_ACCESS_DENIED(),
		ResponseHeaders: &map[string]*string{
			"Access-Control-Allow-Origin":  jsii.String("'*'"),
			"Access-Control-Allow-Headers": jsii.String("'*'"),
		},
		StatusCode: jsii.String("403"),
		RestApi:    api,
	})

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

	pathsWithCors := map[string]bool{}

	var methods []awsapigateway.Method

	for _, v := range options.Integrations {
		_, ok := pathsWithCors[v.Path]
		methods = append(methods, AddLambdaIntegrationToAPIGateway(api, v.Function, v.Path, v.Method, v.Authorizer, !ok)...)
		pathsWithCors[v.Path] = true
	}

	deployment := awsapigateway.NewDeployment(this, jsii.String(fmt.Sprintf("api-gw-deployment-%s", time.Now().String())), &awsapigateway.DeploymentProps{
		Api:         api,
		Description: jsii.String("Deployment"),
	})

	// A deployment is a snapshot of the API taken when it is created, so every method has to
	// exist first. The API here is imported by id rather than owned, which is why this is not
	// automatic: CDK cannot infer that these methods belong to it, so nothing orders the two
	// and CloudFormation is free to create the deployment before the methods.
	//
	// The failure is quiet and only shows up when a route is added — existing routes are
	// already in the previous snapshot, so the stage keeps serving them while the new one
	// returns 403 "Missing Authentication Token" until something triggers a second deploy.
	//
	// The dependency is on the CfnMethod alone, not the method construct. A construct pulls
	// in its whole subtree, and a method's subtree holds the Lambda permission whose
	// SourceArn names the stage — which depends on this deployment, closing a cycle that
	// makes the template undeployable.
	for _, m := range methods {
		if cfnMethod := m.Node().DefaultChild(); cfnMethod != nil {
			deployment.Node().AddDependency(cfnMethod)
		}
	}

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

// AddLambdaIntegrationToAPIGateway returns every method it created, so a caller building a
// deployment can depend on them. Callers that ignore the return are unaffected.
func AddLambdaIntegrationToAPIGateway(api awsapigateway.IRestApi, handler awslambda.IFunction, path, method string, authorizer awsapigateway.IAuthorizer, addCors bool) []awsapigateway.Method {

	integration := awsapigateway.NewLambdaIntegration(handler, &awsapigateway.LambdaIntegrationOptions{})

	resource := api.Root().ResourceForPath(jsii.String(path))

	var methods []awsapigateway.Method

	if addCors {
		methods = append(methods, resource.AddCorsPreflight(&awsapigateway.CorsOptions{
			AllowOrigins:     jsii.Strings("*"),
			AllowCredentials: jsii.Bool(true),
			AllowHeaders:     jsii.Strings("*"),
			AllowMethods:     jsii.Strings("*"),
			StatusCode:       jsii.Number(201),
		}))
	}

	options := &awsapigateway.MethodOptions{}
	if authorizer != nil {
		options.AuthorizationType = "CUSTOM"
		options.Authorizer = authorizer
	}

	methods = append(methods, resource.AddMethod(jsii.String(method), integration, options))

	return methods

}
