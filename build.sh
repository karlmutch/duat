#!/bin/bash
set -e
docker build -t duat-build:latest --build-arg USER=$USER --build-arg USER_ID=`id -u $USER` --build-arg USER_GROUP_ID=`id -g $USER` .
docker run -e GITHUB_TOKEN=$GITHUB_TOKEN -v $GOPATH:/project duat-build:latest
result=$?
if [ $result -ne 0 ]; then
    echo "failed with code $result"
    exit $result
fi
go run ./build.go -image-only -dirs example/artifact
