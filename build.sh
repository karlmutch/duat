#!/bin/bash
docker build -t duat-build:latest --build-arg USER=$USER --build-arg USER_ID=`id -u $USER` --build-arg USER_GROUP_ID=`id -g $USER` .
docker run -e GITHUB_TOKEN=$GITHUB_TOKEN -v $GOPATH:/project duat-build:latest
if [ $? -ne 0 ]; then
    echo "Failure $?"
    exit $?
fi
echo "Done" ; docker container prune -f
