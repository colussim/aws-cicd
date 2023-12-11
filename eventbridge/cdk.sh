#!/bin/sh

if [ "$1" = "deploy" ]; then
    go run main.go -destroy=false
elif [ "$1" = "destroy" ]; then
    go run main.go -destroy=true
else
    echo "Usage: $0 [deploy|destroy]"
    exit 1
fi



