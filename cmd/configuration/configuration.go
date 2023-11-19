package main

import (
	"fmt"
	"github.com/awkhan/go-aws-cdk-constructs/pkg/common"
	"github.com/aws/aws-cdk-go/awscdk/v2"

	"github.com/aws/jsii-runtime-go"
)

type Configuration struct {
	DomainName string `json:"domainName"`
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	var cfg Configuration
	common.ParseConfigurationInto(app, &cfg)

	fmt.Println(cfg)

	app.Synth(nil)
}
