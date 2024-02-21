package common

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"os"
)

var USEastStackProps = &awscdk.StackProps{Env: &awscdk.Environment{Region: jsii.String("us-east-1")}, CrossRegionReferences: jsii.Bool(true)}
var CaCentralStackProps = &awscdk.StackProps{Env: &awscdk.Environment{Region: jsii.String("ca-central-1")}, CrossRegionReferences: jsii.Bool(true)}

func ParseConfigurationInto(app awscdk.App, cfg interface{}) {

	m := app.Node().GetContext(jsii.String(env.(string))).(map[string]interface{})
	data, err := json.Marshal(m)
	if err != nil {
		fmt.Println("unable to parse configuration data")
		os.Exit(2)
	}

	err = json.Unmarshal(data, &cfg)
	if err != nil {
		fmt.Println("unable to unmarshal configuration")
		os.Exit(3)
	}

}
