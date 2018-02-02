#!/bin/bash -x
set -e
./cmd/bump-ver/build.sh
if [ $? -ne 0 ]; then
    echo "bump-ver build failed"
    exit $?
fi
