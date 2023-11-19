package common

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
	"os"
)

func ParseConfigurationInto(app awscdk.App, cfg interface{}) {

	env := app.Node().TryGetContext(jsii.String("environment"))
	if env == nil {
		fmt.Println("no environment supplied. Pass in as '-c environment=production|staging")
		os.Exit(1)
	}

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
