#!/bin/sh

if [ "$1" = "deploy" ]; then
    cdk deploy 
    go run gitdep.go -destroy=false
elif [ "$1" = "destroy" ]; then
    go run gitdep.go -destroy=true
    cdk destroy --force
else
    echo "Usage: $0 [deploy|destroy]"
    exit 1
fi



