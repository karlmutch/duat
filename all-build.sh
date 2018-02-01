#!/bin/bash -x
set -e
./cmd/bump-md-ver/build.sh
if [ $? -ne 0 ]; then
    echo "bump-md-ver build failed"
    exit $?
fi
