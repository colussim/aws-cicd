![AWS](https://img.shields.io/badge/AWS-%23FF9900.svg?style=for-the-badge&logo=amazon-aws&logoColor=white)![Amazon EKS](https://img.shields.io/static/v1?style=for-the-badge&message=Amazon+EKS&color=222222&logo=Amazon+ECS&logoColor=FF9900&label=)![Static Badge](https://img.shields.io/badge/Go-v1.21-blue:) ![Static Badge](https://img.shields.io/badge/AWS_CDK-v2.96.2-blue:)


# Welcome to your CDK Deployment with Go.

The purpose of this deployment is to Create a AWS EKS cluster.


* The `cdk.json` file tells the CDK toolkit how to execute your app.
* The `config.json` Contains the parameters to be initialized to deploy the task :
```
Config.json :

ClusterName:	EKS Cluster Name
Index:          index for Cluster Name
VPCid:          VPC ID
K8sVersion:     Version of Kubernetes: default 1.27
Workernode:     Number of Worker Node        
EksAdminRole:   Name of EKS Role
EBSRole:        Name of EBS Role for storage
Instance:       AWS Instance types using for EKS
InstanceSize:   AWS Instance size
AddonVersion	Addon version for EBS CSI Driver : 1.24.0-eksbuild.1
ScName          Name of the Storage Storrage class use,
ScNamef         Path of store class manifest file : default dist/sc.yaml for addons
```    

> AWS CDK for go currently only supports kubernetes version 1.27.
> You can upgrade to 1.28 directly from **AWS Management Console** or with the **eksctl** command.

## What does this task do?

- Create the different roles needed for EKS
- Create a EKS Cluster with LoadBalancer services
- Add Add-ons : EBS CSI Driver and deployed Manifest : create Storage Class

## Useful commands

 * `cdk deploy`      deploy this stack to your default AWS account/region
 * `cdk destroy`     cleaning up stack

## ✅ Setup Environment

Run the following command to automatically install all the required modules based on the go.mod and go.sum files:

```bash
aws-cicd:/eks/> go mod download

``` 

## ✅ Deploying your cluster

Let’s deploy a cluster! When you’re ready, run **cdk deploy**

```bash
aws-cicd:/eks/> cdk deploy

``` 

❗️ When you run the deployment command, you're likely to get an warning like this: 

``` 
Warning at /EksStack03/ClustWorkshop03] Could not auto-tag private subnet subnet-0359e8e6d390220ed with "kubernetes.io/role/internal-elb=1", please remember to do this manually [ack: @aws-cdk/aws-eks:clusterMustManuallyTagSubnet]
[Warning at /EksStack03/ClustWorkshop03] Could not auto-tag private subnet subnet-09b7b15a2a6f207fa with "kubernetes.io/role/internal-elb=1", please remember to do this manually [ack: @aws-cdk/aws-eks:clusterMustManuallyTagSubnet]
[Warning at /EksStack03/ClustWorkshop03] Could not auto-tag public subnet subnet-0ddbf63f1465fd5f9 with "kubernetes.io/role/elb=1", please remember to do this manually [ack: @aws-cdk/aws-eks:clusterMustManuallyTagSubnet]
[Warning at /EksStack03/ClustWorkshop03] Could not auto-tag public subnet subnet-05673dfe2e1a99d8e with "kubernetes.io/role/elb=1", please remember to do this manually [ack: @aws-cdk/aws-eks:clusterMustManuallyTagSubnet]
``` 

Ignore it,the problem is that at this stage of deployment, the CDK is unable to check whether the right tags are present in the subnets.
If you've skipped the previous step of creating the VPC, the tags are set on each subnet, otherwise you'll have to set them manually via the AWS console.


❗️ and error like this:
``` 

Error occurred while monitoring stack: Error [ValidationError]: 2 validation errors detected: Value '' at 'stackName' failed to satisfy constraint: Member must have length greater than or equal to 1; Value '' at 'stackName' failed to satisfy constraint: Member must satisfy regular expression pattern: [a-zA-Z][-a-zA-Z0-9]*|arn:[-a-zA-Z0-9:/._+]*

``` 

Ignore it, it's a bug in the CDK version (2.102.0 (build 2abc59a)).
The actual cause for the test failure  was due to my temporary SSO credentials expiring. It seems that when we run this test in our integration environment, we get the printed out errors you see about the invalid stackName .. but those do not actually cause a failure, they are just noise. So this is still a bug, but it's less critical.
For the warning when running synth, we don't explicitly query AWS to find the actual state of the subnets. This warning for all imported subnets simply because we have no way of guaranteeing its tagged correctly, so we figured its better to assume its not.

Not sure there is anything we can do here apart from removing the warning altogether, which we don't want to.

In a few minutes your EKS cluster is up 😀.
After creating your cluster, CDK should output a command that will add your new cluster to your kubeconfig. In this case, it would be:

``` 
 ✨  Deployment time: 1018.37s

Outputs:
✅  EksStack01.ClustWorkshopConfigCommandFAA0F346 = aws eks update-kubeconfig --name ClustWorkshop --region eu-central-1 --role-arn arn:aws:iam::XXXXXX:role/ClustWorkshop-01-AdminRole

✨  Total time: 1023.36s
``` 

✅ Run this command (XXXXXX is your AWS account number):

``` 
CDK:/EKS/> aws eks update-kubeconfig --name ClustWorkshop-01 --region eu-central-1 --role-arn arn:aws:iam::XXXXXX:role/ClustWorkshop-01-AdminRole

``` 
Let’s run kubectl get nodes to get the node in your EKS Cluster and check the connection to your cluster :

```bash 
aws-cicd:/eks/> kubectl get nodes
NAME                                              STATUS   ROLES    AGE   VERSION
ip-192-168-141-83.eu-central-1.compute.internal   Ready    <none>   16m   v1.27.5-eks-43840fb
ip-192-168-206-33.eu-central-1.compute.internal   Ready    <none>   16m   v1.27.5-eks-43840fb
``` 

## ✅ Adding Add-ons

Not only can you create EKS clusters in CDK, but you can also deploy Add-on The EBS CSI Driver and new storage class (managed-csi)
You need run the following command to automatically install all the required modules based on the go.mod and go.sum files:

```bash
Caws-cicd:/eks/> cd addons
```

Run the following command to automatically install all the required modules based on the go.mod and go.sum files:

```bash
aws-cicd:/eks/addons> go mod download
``` 
Run Add-ons deployment :

```bash 
aws-cicd:/eks/addons> cdk deploy --context destroy=false
```

We can check the add-on EBS-CSI driver is actived :
```bash
aws-cicd:/eks/addons> kubectl get deployment/ebs-csi-controller -n kube-system
NAME                 READY   UP-TO-DATE   AVAILABLE   AGE
ebs-csi-controller   2/2     2            2           1m00s
``` 


We can check if storage class **managed-csi** is created  :
```bash 
aws-cicd:/eks/addons> kubectl get sc
NAME            PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
gp2 (default)   kubernetes.io/aws-ebs   Delete          WaitForFirstConsumer   false                  12h
managed-csi     ebs.csi.aws.com         Delete          WaitForFirstConsumer   false                  1m56s

``` 

Now 😀 all set for SonarQube deployment 

Nest step : Deployment Sonarqube

----

## ✅ Ressources

▶️ [EBS CSI driver add-on](https://docs.aws.amazon.com/eks/latest/userguide/managing-ebs-csi.html)

-----
<table>
<tr style="border: 0px transparent">
	<td style="border: 0px transparent"> <a href="../vpc/README.md" title="Creating a VPC">⬅ Previous</a></td><td style="border: 0px transparent"><a href="../sonarqube/README.md" title="SonarQube deployment">Next ➡</a></td><td style="border: 0px transparent"><a href="../README.md" title="home">🏠</a></td>
</tr>
<tr style="border: 0px transparent">
<td style="border: 0px transparent">Creating a VPC</td><td style="border: 0px transparent">SonarQube deployment</td><td style="border: 0px transparent"></td>
</tr>

</table>
