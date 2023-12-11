![AWS](https://img.shields.io/badge/AWS-%23FF9900.svg?style=for-the-badge&logo=amazon-aws&logoColor=white)![Amazon EKS](https://img.shields.io/static/v1?style=for-the-badge&message=Amazon+EKS&color=222222&logo=Amazon+ECS&logoColor=FF9900&label=)![Static Badge](https://img.shields.io/badge/Go-v1.21-blue:) ![Static Badge](https://img.shields.io/badge/AWS_CDK-v2.96.2-blue:)


# Welcome to your CDK Deployment with Go.

The purpose of this deployment is to Adding Add-ons in AWS EKS Cluster :
- EBS CSI Driver
- Storage class add label worker on AWS EKS Nodes

The `cdk.json` file tells the CDK toolkit how to execute your app.

## Useful commands

 * `cdk deploy --context destroy=false` deploy this stack to your default AWS account/region
 * `cdk diff`          compare deployed stack with current state
 * `cdk synth`         emits the synthesized CloudFormation template
 * `go test`           run unit tests
 * `cdk destroy --context destroy=true` cleaning up stack

 ## ✅ Setup Environment

Run the following command to automatically install all the required modules based on the go.mod and go.sum files:

```bash
CDK:/EKS/Eksstackconfig/> go mod download
``` 
