package main

import (
	"fmt"
	"github.com/awkhan/go-aws-cdk-constructs/pkg/hostedzone"
	"github.com/awkhan/go-aws-cdk-constructs/pkg/website"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"os"

	"github.com/aws/jsii-runtime-go"
)

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	zoneStack := awscdk.NewStack(app, jsii.String("hosted-zone-certificate"), &awscdk.StackProps{Env: &awscdk.Environment{Region: jsii.String("us-east-1")}, CrossRegionReferences: jsii.Bool(true)})
	z := hostedzone.New(zoneStack, "hosted-zone", hostedzone.Options{Name: "sandbox.itinto.com", CreateCertificate: true})

	websiteStack := awscdk.NewStack(app, jsii.String("website"), &awscdk.StackProps{Env: &awscdk.Environment{Region: jsii.String("ca-central-1")}, CrossRegionReferences: jsii.Bool(true)})
	assetDir := createTemporaryAssets()
	website.New(websiteStack, "website", website.Options{DomainName: "sandbox.itinto.com", AssetPath: assetDir, Certificate: *z.Certificate, HostedZone: z.HostedZone})

	os.RemoveAll(assetDir)

	app.Synth(nil)
}

func createTemporaryAssets() string {
	err := os.Mkdir("build", 0777)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("build/index.html", []byte("<body>Hello World 5</body>"), 0644)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("build/asset.json", []byte("{}"), 0644)
	if err != nil {
		panic(err)
	}

	d, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	path := fmt.Sprintf("%s/build", d)

	fmt.Println(path)

	return path

}
