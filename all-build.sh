#!/bin/bash -x
set -e
./cmd/semver/build.sh
if [ $? -ne 0 ]; then
    echo "semver build failed"
    exit $?
fi
