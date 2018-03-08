#!/bin/bash -x
set -e
./cmd/semver/build.sh
result=$?
if [ $result -ne 0 ]; then
    echo "semver build failed"
    exit $result
fi
example/artifact/build.go
result=$?
if [ $result -ne 0 ]; then
    echo "example artifact build failed"
    exit $result
fi
