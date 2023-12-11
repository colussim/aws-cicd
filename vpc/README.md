![AWS](https://img.shields.io/badge/AWS-%23FF9900.svg?style=for-the-badge&logo=amazon-aws&logoColor=white)![Static Badge](https://img.shields.io/badge/Go-v1.21-blue:) ![Static Badge](https://img.shields.io/badge/AWS_CDK-v2.96.2-blue:)


# Welcome to your CDK Deployment with Go.

The purpose of this deployment is to Create a VPC and subnets.
It adds tags to the subnets needed to deploy AWS EKS.


* The `cdk.json` file tells the CDK toolkit how to execute your app.
* The `config.json` Contains the parameters to be initialized to deploy the task :
```
Config.json :

	VPCName:	the VPC Name
	VPCcidr:	CIDR Network
	ZA:			Number of Availability zone (minimum 2)
	SGName:		Security Group Name
	SGDescription:	Security Group Desciption	        
```    

## What does this task do?

- Create a VPC
- create a Security Groupe

## Useful commands

 * `cdk deploy`      deploy this stack to your default AWS account/region
 * `cdk destroy`     cleaning up

 ## Setup Environment

Run the following command to automatically install all the required modules based on the go.mod and go.sum files:

```bash
aws-cicd:/vpc/> go mod download
``` 

## Deploying your VPC

Let’s deploy a VPC! When you’re ready, run **cdk deploy**

``` bash
aws-cicd:/vpc/> cdk deploy

VPCStack01: deploying... [1/1]
VPCStack01: creating CloudFormation changeset...

 ✅  VPCStack01

✨  Deployment time: 199.91s

Outputs:
VPCStack01.VPCCREATED = vpc-07126b9cc1c292878
Stack ARN:
arn:aws:cloudformation:eu-central-1:103078382956:stack/VPCStack01/37a5c9d0-78c0-11ee-86f1-02749d6c9b45

✨  Total time: 206.25s


``` 
-----
<table>
<tr style="border: 0px transparent">
	<td style="border: 0px transparent"> <a href="../README.md" title="Introduction">⬅ Previous</a></td><td style="border: 0px transparent"><a href="../eks/README.md" title="Creating a EKS Cluster">Next ➡</a></td>
</tr>
<tr style="border: 0px transparent">
<td style="border: 0px transparent">Introduction</td><td style="border: 0px transparent">Creating a EKS Cluster</td>
</tr>

</table>
