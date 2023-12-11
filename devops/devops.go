package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscodebuild"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscodecommit"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscodepipeline"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscodepipelineactions"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssecretsmanager"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/constructs-go/constructs/v10"

	"github.com/aws/jsii-runtime-go"
)

type DevopsStackProps struct {
	awscdk.StackProps
}

type ConfAuth struct {
	Region     string
	Account    string
	SSOProfile string
	Index      string
	AWSsecret  string
}

type Configuration struct {
	Reponame     string
	Desc         string
	GitRepo      string
	Recr         string
	ImgTag       string
	BuildPr      string
	PiplineN     string
	ClusterName  string
	EksAdminRole string
	Platform     string
}

func GetConfig(configcrd ConfAuth, configjs Configuration) (ConfAuth, Configuration) {

	fconfig, err := os.ReadFile("config.json")
	if err != nil {
		panic("❌ Problem with the configuration file : config.json")
		os.Exit(1)
	}
	if err := json.Unmarshal(fconfig, &configjs); err != nil {
		fmt.Println("❌ Error unmarshaling JSON:", err)
		os.Exit(1)
	}

	fconfig2, err := os.ReadFile("../config_crd.json")
	if err != nil {
		panic("❌ Problem with the configuration file : config_crd.json")
		os.Exit(1)
	}
	if err := json.Unmarshal(fconfig2, &configcrd); err != nil {
		fmt.Println("❌ Error unmarshaling JSON:", err)
		os.Exit(1)
	}
	return configcrd, configjs
}

func NewDevopsStack(scope constructs.Construct, id string, props *DevopsStackProps, AppConfig Configuration, AppConfig1 ConfAuth) awscdk.Stack {
	var sprops awscdk.StackProps
	var buildImage awscodebuild.IBuildImage
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	os.Setenv("AWS_SDK_LOAD_CONFIG", "true")
	os.Setenv("AWS_PROFILE", AppConfig1.SSOProfile)
	StackRepoN := "CodeCommitRepo" + AppConfig1.Index
	BuilRole := "BuildAdminRole" + AppConfig1.Index
	BuildPrName := AppConfig.BuildPr + "-" + AppConfig1.Index
	PiplineName1 := AppConfig.PiplineN + "-" + AppConfig1.Index
	ERCReposName := AppConfig.Recr + "-" + AppConfig1.Index
	RepoNameCd := AppConfig.Reponame + "-" + AppConfig1.Index

	if AppConfig.Platform == "x86" {
		buildImage = awscodebuild.LinuxBuildImage_AMAZON_LINUX_2_5()
	} else {
		buildImage = awscodebuild.LinuxBuildImage_AMAZON_LINUX_2_ARM_3()
	}

	// Create a Build Admin Role
	buildAdminRole := awsiam.NewRole(stack, &BuilRole, &awsiam.RoleProps{
		AssumedBy:   awsiam.NewServicePrincipal(jsii.String("codebuild.amazonaws.com"), nil),
		Description: jsii.String("IAM Role for CodeBuild"),
		RoleName:    &BuilRole,
	})
	buildAdminRole.AddManagedPolicy(awsiam.ManagedPolicy_FromAwsManagedPolicyName(aws.String("AmazonEKSClusterPolicy")))

	// Create a CodeCommit repository
	Repo := awscodecommit.NewRepository(stack, &StackRepoN, &awscodecommit.RepositoryProps{
		RepositoryName: &RepoNameCd,
		Description:    &AppConfig.Desc,
	})

	// Create an Amazon ECR repository
	awsecr.NewRepository(stack, &ERCReposName, &awsecr.RepositoryProps{
		RepositoryName:   &ERCReposName,
		RemovalPolicy:    awscdk.RemovalPolicy_DESTROY,
		AutoDeleteImages: jsii.Bool(true),
	})

	// Define a CodeBuild project
	awscodebuild.NewProject(stack, &AppConfig.BuildPr, &awscodebuild.ProjectProps{
		Source: awscodebuild.Source_CodeCommit(&awscodebuild.CodeCommitSourceProps{
			Repository: Repo,
		}),
		ProjectName: &BuildPrName,
		Role:        buildAdminRole,
		Environment: &awscodebuild.BuildEnvironment{
			BuildImage: buildImage,
			Privileged: jsii.Bool(true),
		},
		EnvironmentVariables: &map[string]*awscodebuild.BuildEnvironmentVariable{
			"AWS_ACCOUNT_ID": &awscodebuild.BuildEnvironmentVariable{
				Value: &AppConfig1.Account,
			},
			"IMAGE_TAG": &awscodebuild.BuildEnvironmentVariable{
				Value: &AppConfig.ImgTag,
			},
			"IMAGE_REPO_NAME": &awscodebuild.BuildEnvironmentVariable{
				Value: &ERCReposName,
			},
		},
	})

	//	Bproject.Node().AddDependency(buildAdminRole)

	// Get Sonar Secret ARN
	secretName := AppConfig1.AWSsecret + AppConfig1.Index
	secret := awssecretsmanager.Secret_FromSecretNameV2(stack, jsii.String("ExistingSecret"), &secretName)
	secretValue0 := *secret.SecretArn()
	SecretValue := secretValue0

	// Create an inline policy
	PolicyStat1 := awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{

		Actions: &[]*string{
			jsii.String("ecr:BatchCheckLayerAvailability"),
			jsii.String("ecr:CompleteLayerUpload"),
			jsii.String("ecr:GetAuthorizationToken"),
			jsii.String("ecr:InitiateLayerUpload"),
			jsii.String("ecr:PutImage"),
			jsii.String("ecr:UploadLayerPart"),
			jsii.String("eks:*"),
			jsii.String("s3:*"),
			jsii.String("secretsmanager:GetResourcePolicy"),
			jsii.String("secretsmanager:GetSecretValue"),
			jsii.String("secretsmanager:DescribeSecret"),
			jsii.String("secretsmanager:ListSecretVersionIds"),
			jsii.String("secretsmanager:ListSecrets"),
			jsii.String("kms:*"),
			jsii.String("sts:AssumeRole"),
		},
		Resources: &[]*string{
			jsii.String("*"),
			jsii.String(SecretValue),
		},
		Effect: awsiam.Effect_ALLOW,
	})
	buildAdminRole.AddToPolicy(PolicyStat1)

	// Create the source artifact
	sourceArtifact := awscodepipeline.NewArtifact(jsii.String("SourceArtifacts"))

	// Create the pipeline
	pipeline := awscodepipeline.NewPipeline(stack, &PiplineName1, &awscodepipeline.PipelineProps{
		EnableKeyRotation: jsii.Bool(true),
		PipelineName:      &PiplineName1,
	})

	// Define the source stage
	sourceStage := pipeline.AddStage(&awscodepipeline.StageOptions{
		StageName: jsii.String("SourceStage"),
	})
	sourceAction := awscodepipelineactions.NewCodeCommitSourceAction(&awscodepipelineactions.CodeCommitSourceActionProps{
		ActionName:         jsii.String("Source"),
		Output:             sourceArtifact,
		Repository:         awscodecommit.Repository_FromRepositoryName(stack, &RepoNameCd, &RepoNameCd),
		VariablesNamespace: jsii.String("SourceVariables"),
	})

	sourceStage.AddAction(sourceAction)

	// Define the build stage
	buildStage := pipeline.AddStage(&awscodepipeline.StageOptions{
		StageName: jsii.String("BuildStage"),
	})
	buildAction := awscodepipelineactions.NewCodeBuildAction(&awscodepipelineactions.CodeBuildActionProps{
		ActionName: jsii.String("Build"),
		Input:      sourceArtifact,
		Project:    awscodebuild.Project_FromProjectName(stack, &BuildPrName, &BuildPrName),
		EnvironmentVariables: &map[string]*awscodebuild.BuildEnvironmentVariable{
			"SourceBranch": {
				Value: jsii.String("#{SourceVariables.BranchName}"),
				Type:  awscodebuild.BuildEnvironmentVariableType_PLAINTEXT,
			},
		},
		VariablesNamespace: jsii.String("BuildVariables"),
	})

	buildStage.AddAction(buildAction)

	ArnBuildRole := *buildAdminRole.RoleArn()

	// Output the bucket name
	awscdk.NewCfnOutput(stack, jsii.String("ARN Role BuildProject"), &awscdk.CfnOutputProps{
		Value: &ArnBuildRole,
	})

	return stack
}

func main() {
	defer jsii.Close()

	// Read configuration from config.json file
	var configcrd ConfAuth
	var config1 Configuration
	var AppConfig1, AppConfig = GetConfig(configcrd, config1)
	Stack1 := "DevopsStack" + AppConfig1.Index

	app := awscdk.NewApp(nil)

	NewDevopsStack(app, Stack1, &DevopsStackProps{
		awscdk.StackProps{
			Env: env(AppConfig1.Region, AppConfig1.Account),
		},
	}, AppConfig, AppConfig1)

	app.Synth(nil)

}

func env(Region1 string, Account1 string) *awscdk.Environment {

	return &awscdk.Environment{
		Account: &Account1,
		Region:  &Region1,
	}

}
