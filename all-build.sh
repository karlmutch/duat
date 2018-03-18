#!/bin/bash -x
set -e
go run ./build.go -module cmd -r
result=$?
if [ $result -ne 0 ]; then
    exit $result
fi
go run ./build.go -module example/artifact
result=$?
if [ $result -ne 0 ]; then
    exit $result
fi
