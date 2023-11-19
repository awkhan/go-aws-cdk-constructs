package hostedzone

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type Options struct {
	awscdk.StackProps
	Name              string
	CreateCertificate bool
}

type HostedZone struct {
	constructs.Construct
	HostedZone  awsroute53.IHostedZone
	Certificate *awscertificatemanager.ICertificate
}

func New(scope constructs.Construct, id string, options Options) HostedZone {

	this := constructs.NewConstruct(scope, &id)
	hostedZone := awsroute53.NewHostedZone(this, jsii.String("hosted-zone"), &awsroute53.HostedZoneProps{
		ZoneName: jsii.String(options.Name),
	})

	result := HostedZone{Construct: this, HostedZone: hostedZone}

	var certificate awscertificatemanager.ICertificate
	if options.CreateCertificate {
		certificate = awscertificatemanager.NewCertificate(this, jsii.String("certificate"), &awscertificatemanager.CertificateProps{
			DomainName:      jsii.String(options.Name),
			CertificateName: jsii.String(options.Name),
			Validation:      awscertificatemanager.CertificateValidation_FromDns(hostedZone),
		})
		result.Certificate = &certificate
	}

	return result

}
