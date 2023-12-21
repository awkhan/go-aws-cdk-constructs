package githuboidcprovider

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type Options struct {
	awscdk.StackProps
	GitHubRepoPath string
}

type GitHubOIDCProvider struct {
	constructs.Construct
}

const (
	clientID   = "sts.amazonaws.com"
	githubHost = "token.actions.githubusercontent.com"
)

func New(scope constructs.Construct, id string, options Options) GitHubOIDCProvider {

	this := constructs.NewConstruct(scope, &id)

	provider := awsiam.NewOpenIdConnectProvider(this, jsii.String("oidc-provider"), &awsiam.OpenIdConnectProviderProps{
		Url:         jsii.String("https://" + githubHost),
		ClientIds:   jsii.Strings(clientID),
		Thumbprints: jsii.Strings("6938fd4d98bab03faadb97b34396831e3780aea1", "1c58a3a8518e8759bf075b76b750d4f2df264fcd"),
	})

	conditions := map[string]interface{}{
		"StringEquals": map[string]string{
			githubHost + ":aud": clientID,
		},
		"StringLike": map[string]string{
			githubHost + ":sub": "repo:" + options.GitHubRepoPath,
		},
	}

	policies := []awsiam.IManagedPolicy{
		awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AdministratorAccess")),
	}

	awsiam.NewRole(this, jsii.String("deployment-role"), &awsiam.RoleProps{
		AssumedBy:          awsiam.NewWebIdentityPrincipal(provider.OpenIdConnectProviderArn(), &conditions),
		Description:        jsii.String("Used to deploy from GitHub actions"),
		MaxSessionDuration: awscdk.Duration_Hours(jsii.Number(1)),
		RoleName:           jsii.String("ITINTOGitHubDeploy"),
		ManagedPolicies:    &policies,
	})

	return GitHubOIDCProvider{this}

}
