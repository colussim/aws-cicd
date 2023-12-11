#!/bin/sh

# Reset workshop

cd eventbridge
./cdk.sh destroy
cd ../devops
./cdk.sh destroy
cd ../sonarqube
./cdk.sh destroy
cd ../eks
cdk destroy
