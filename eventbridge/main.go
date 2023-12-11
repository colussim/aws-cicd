package main

import (
	"CDK/pkg/mainconfig"

	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	evtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type Configuration struct {
	Reponame         string
	Desc             string
	GitRepo          string
	Recr             string
	ImgTag           string
	BuildPr          string
	PiplineN         string
	ClusterName      string
	EksAdminRole     string
	SecondBramchName string
}

func readJSONConfig(filename string, config interface{}) {
	fconfig, err := os.ReadFile(filename)
	if err != nil {
		panic(fmt.Sprintf("❌ Problem with the configuration file: %s", filename))
		os.Exit(1)
	}
	if err := json.Unmarshal(fconfig, config); err != nil {
		fmt.Println("❌ Error unmarshaling JSON:", err)
		os.Exit(1)
	}
}

func GetConfig(configcrd mainconfig.ConfAuth, configjs Configuration) (mainconfig.ConfAuth, Configuration) {

	readJSONConfig("../devops/config.json", &configjs)
	readJSONConfig("../config_crd.json", &configcrd)

	return configcrd, configjs
}

func getAssumeRolePolicyDocument() string {
	return strings.TrimSpace(`
{
  "Version": "2012-10-17",
  "Statement": [
   {
      "Effect": "Allow",
      "Principal": {
      "Service": "codebuild.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
`)
}

func createIAMRole(ctx context.Context, roleName string, cfg aws.Config) (string, error) {
	// Create IAM client
	iamClient := iam.NewFromConfig(cfg)

	// Define IAM role policy document
	policyDocument := strings.TrimSpace(`
{
    "Version": "2012-10-17",
    "Statement": [
     {
       "Effect": "Allow",
       "Action": "codebuild:StartBuild",
       "Resource": "*"
      }
    ]
}
`)

	// Create IAM role
	createRoleOutput, err := iamClient.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(getAssumeRolePolicyDocument()),
	})
	if err != nil {
		return "", err
	}

	// Attach the policy to the role
	_, err = iamClient.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyName:     aws.String("EventBridgeCodeBuildPolicy"),
		PolicyDocument: aws.String(policyDocument),
	})
	if err != nil {
		return "", err
	}

	for {
		getRoleOutput, err := iamClient.GetRole(ctx, &iam.GetRoleInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			return "", err
		}

		if aws.ToString(getRoleOutput.Role.Arn) != "" {
			return aws.ToString(getRoleOutput.Role.Arn), nil
		}

		time.Sleep(5 * time.Second)
	}

	roleArn := createRoleOutput.Role.Arn
	fmt.Printf("❌ IAM Role ARN: %s\n", *roleArn)

	return *roleArn, nil
}

func createEventBridgeRule(ctx context.Context, ruleName, roleArn, eventRuleArnVariable, codeCommitRepoArn string, cfg aws.Config) (string, error) {
	// Create EventBridge client
	eventBridgeClient := eventbridge.NewFromConfig(cfg)

	// Define EventBridge rule input transformer template
	inputTransformerTemplate := strings.TrimSpace(`
	{
		"environmentVariablesOverride": [
		  {
			"name": "SourceBranch",
			"type": "PLAINTEXT",
			"value": "<SourceBranch>"
		  },
		  {
			"name": "DestinationBranch",
			"type": "PLAINTEXT",
			"value": "<DestinationBranch>"
		  },
		  {
			"name": "PRKey",
			"type": "PLAINTEXT",
			"value": "<PRKey>"
		  }
		],
		"sourceVersion": "<sourceReference>"
	  }
	  `)

	// Replace placeholders in the input transformer template
	inputTransformerTemplate = replacePlaceholders(inputTransformerTemplate, map[string]string{
		"<SourceBranch>":      "$.detail.destinationReference",
		"<DestinationBranch>": "$.detail.pullRequestId",
		"<PRKey>":             "$.detail.sourceReference",
		"<sourceReference>":   "$.detail.sourceReference",
	})

	// Create EventBridge rule
	createRuleOutput, err := eventBridgeClient.PutRule(ctx, &eventbridge.PutRuleInput{
		Name:         aws.String(ruleName),
		EventPattern: aws.String(fmt.Sprintf(`{"detail-type":["CodeCommit Pull Request State Change"],"resources":["%s"],"source":["aws.codecommit"]}`, codeCommitRepoArn)),
		State:        evtypes.RuleStateEnabled,
	})
	if err != nil {
		return "", err
	}

	// Create an EventBridge target
	_, err = eventBridgeClient.PutTargets(ctx, &eventbridge.PutTargetsInput{
		Rule: aws.String(ruleName),
		Targets: []evtypes.Target{
			{
				Id:      aws.String("SonarCodeBuildProject"),
				Arn:     aws.String(eventRuleArnVariable),
				RoleArn: aws.String(roleArn),
				InputTransformer: &evtypes.InputTransformer{
					InputPathsMap: map[string]string{
						"DestinationBranch": "$.detail.destinationReference",
						"PRKey":             "$.detail.pullRequestId",
						"SourceBranch":      "$.detail.sourceReference",
						"sourceReference":   "$.detail.sourceReference",
					},
					InputTemplate: aws.String(inputTransformerTemplate),
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	// Wait for EventBridge rule to be created
	time.Sleep(5 * time.Second)

	eventRuleArn := createRuleOutput.RuleArn

	return *eventRuleArn, nil

}

func replacePlaceholders(template string, replacements map[string]string) string {
	for placeholder, replacement := range replacements {
		template = strings.ReplaceAll(template, placeholder, replacement)
	}
	return template
}

func deleteEventBridgeRule(ctx context.Context, ruleName string) error {
	// Create EventBridge client

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	eventBridgeClient := eventbridge.NewFromConfig(cfg)

	// Remove targets from the rule
	_, err = eventBridgeClient.RemoveTargets(ctx, &eventbridge.RemoveTargetsInput{
		Ids:  []string{"SonarCodeBuildProject"},
		Rule: &ruleName,
	})
	if err != nil {
		return err
	}

	// Delete the rule
	_, err = eventBridgeClient.DeleteRule(ctx, &eventbridge.DeleteRuleInput{
		Name: &ruleName,
	})
	if err != nil {
		return err
	}

	return nil
}

func deleteIAMRole(ctx context.Context, roleName string) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	// Create IAM client
	iamClient := iam.NewFromConfig(cfg)

	// Detach and delete the role policy
	_, err = iamClient.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
		RoleName:   &roleName,
		PolicyName: aws.String("EventBridgeCodeBuildPolicy"),
	})
	if err != nil {
		return err
	}

	// Delete IAM role
	_, err = iamClient.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: &roleName,
	})
	if err != nil {
		return err
	}
	return nil
}

func main() {

	var configcrd mainconfig.ConfAuth
	var config1 Configuration
	var AppConfig1, AppConfig = GetConfig(configcrd, config1)

	eventBridgeRuleArnVariable := "arn:aws:codebuild:" + AppConfig1.Region + ":" + AppConfig1.Account + ":project/" + AppConfig.BuildPr + "-" + AppConfig1.Index
	codeCommitRepoArn := "arn:aws:codecommit:" + AppConfig1.Region + ":" + AppConfig1.Account + ":" + AppConfig.Reponame + "-" + AppConfig1.Index
	ruleName := "OnPullRequestSonarTrigger-" + AppConfig1.Index
	roleName := "EventBridgeCodeBuildRole-" + AppConfig1.Index

	os.Setenv("AWS_SDK_LOAD_CONFIG", "true")
	os.Setenv("AWS_PROFILE", "default")

	ctx := context.TODO()

	destroyFlag := flag.Bool("destroy", false, "Set to true to destroy the added statement in the trust policy")
	// Parse the command-line arguments
	flag.Parse()

	// Load AWS onfiguration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("❌ unable to load config, %v", err)
	}

	if *destroyFlag {
		// Delete IAM role
		err = deleteIAMRole(ctx, roleName)
		if err != nil {
			log.Fatalf("❌ unable to delete IAM role, %v", err)
		}

		fmt.Printf("✅ IAM role '%s' deleted successfully.\n", roleName)

		// Delete EventBridge rule
		err = deleteEventBridgeRule(ctx, ruleName)
		if err != nil {
			log.Fatalf("❌ unable to delete EventBridge rule, %v", err)
		}

		fmt.Printf("✅ EventBridge rule '%s' deleted successfully.\n", ruleName)

	} else {

		// Create an IAM role

		roleArn, err := createIAMRole(ctx, roleName, cfg)
		if err != nil {
			log.Fatalf("❌ unable to create IAM role, %v", err)
		}

		eventRuleArn, err := createEventBridgeRule(ctx, ruleName, roleArn, eventBridgeRuleArnVariable, codeCommitRepoArn, cfg)
		if err != nil {
			log.Fatalf("❌ unable to create EventBridge rule, %v", err)
		}

		fmt.Printf("✅  EventBridge Rule ARN  '%s' created successfully\n", eventRuleArn)
	}
}
