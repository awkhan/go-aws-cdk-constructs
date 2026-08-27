package apigateway

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/jsii-runtime-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deploymentTemplate(t *testing.T) map[string]interface{} {

	t.Helper()

	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("test"), &awscdk.StackProps{
		Env: &awscdk.Environment{Region: jsii.String("ca-central-1"), Account: jsii.String("000000000000")},
	})

	zone := awsroute53.NewPublicHostedZone(stack, jsii.String("zone"), &awsroute53.PublicHostedZoneProps{
		ZoneName: jsii.String("example.com"),
	})

	certificate := awscertificatemanager.NewCertificate(stack, jsii.String("certificate"), &awscertificatemanager.CertificateProps{
		DomainName: jsii.String("api.example.com"),
	})

	// Inline code keeps the fixture free of an asset directory; the runtime is irrelevant to
	// what is being asserted, and provided.al2023 does not accept inline.
	handler := awslambda.NewFunction(stack, jsii.String("handler"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_NODEJS_20_X(),
		Handler: jsii.String("index.handler"),
		Code:    awslambda.Code_FromInline(jsii.String("exports.handler = async () => {};")),
	})

	gateway := New(stack, "api", Options{APIName: "test-api"})

	NewDeployment(stack, "deployment", DeploymentOptions{
		Certificate:  certificate,
		HostedZone:   zone,
		RestAPI:      gateway.API,
		Integrations: []LambdaIntegration{{Function: handler, Path: "thing", Method: "POST"}},
	})

	var template map[string]interface{}
	raw, err := json.Marshal(assertions.Template_FromStack(stack, nil).ToJSON())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &template))

	return template

}

func resourcesOfType(template map[string]interface{}, resourceType string) map[string]map[string]interface{} {

	found := map[string]map[string]interface{}{}

	resources, _ := template["Resources"].(map[string]interface{})
	for id, r := range resources {
		resource, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if resource["Type"] == resourceType {
			found[id] = resource
		}
	}

	return found

}

// The deployment is a snapshot of the API taken when it is created, so CloudFormation has to
// create every method first. The API is imported by id rather than owned, so CDK cannot infer
// that ordering — without an explicit dependency the stage serves a snapshot that predates
// the methods, and a newly added route answers 403 until the next deploy.
func TestDeploymentDependsOnEveryMethod(t *testing.T) {

	template := deploymentTemplate(t)

	deployments := resourcesOfType(template, "AWS::ApiGateway::Deployment")
	require.Len(t, deployments, 1)

	// Every method this deployment is responsible for: the integration's POST and the CORS
	// preflight beside it. The root ANY is excluded — New creates it on the API itself, which
	// in real use is a different stack, so the cross-stack reference already orders it.
	methods := map[string]map[string]interface{}{}
	for id, method := range resourcesOfType(template, "AWS::ApiGateway::Method") {
		properties, _ := method["Properties"].(map[string]interface{})
		if properties["HttpMethod"] == "ANY" {
			continue
		}
		methods[id] = method
	}
	require.Len(t, methods, 2, "the fixture should have produced POST and the CORS OPTIONS")

	var dependsOn []string
	for _, deployment := range deployments {
		raw, ok := deployment["DependsOn"].([]interface{})
		require.True(t, ok, "the deployment declares no DependsOn at all")
		for _, d := range raw {
			dependsOn = append(dependsOn, d.(string))
		}
	}

	for id := range methods {
		assert.Contains(t, dependsOn, id, "deployment must be created after method %s", id)
	}

}

// The CORS preflight is a method like any other and is just as invisible if the snapshot
// predates it, which is what makes a new route's preflight fail alongside it.
func TestDeploymentDependsOnTheCorsPreflight(t *testing.T) {

	template := deploymentTemplate(t)

	var preflights int
	for _, method := range resourcesOfType(template, "AWS::ApiGateway::Method") {
		properties, _ := method["Properties"].(map[string]interface{})
		if properties["HttpMethod"] == "OPTIONS" {
			preflights++
		}
	}

	assert.Equal(t, 1, preflights, "the fixture should have produced exactly one CORS preflight")

}
