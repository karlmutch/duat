#!/bin/bash

if ( find /project -maxdepth 0 -empty | read v );
then
  experiment "source code must be mounted into the /project directory"
  exit 990
fi

set -e
set -o pipefail

export HASH=`git rev-parse HEAD`
export DATE=`date '+%Y-%m-%d_%H:%M:%S%z'`
export PATH=$PATH:$GOPATH/bin
go get -u -f github.com/golang/dep/cmd/dep
go get -u -f github.com/aktau/github-release
export SEMVER=`bump-ver -f ./README.md extract`
TAG_PARTS=$(echo $SEMVER | sed "s/-/\n-/g" | sed "s/\./\n\./g" | sed "s/+/\n+/g")
PRE_RELEASE=""
for part in $TAG_PARTS
do
    start=`echo "$part" | cut -c1-1`
    if [ "$start" == "+" ]; then
        break
    fi
    if [ "$start" == "-" ]; then
        PRE_RELEASE+=$part
    fi
done
flags='-X github.com/karlmutch/duat/version.BuildTime=$DATE -X github.com/karlmutch/duat/version.GitHash=$HASH -X github.com/karlmutch/duat/version.SemVer="${SEMVER}"'
flags="$(eval echo $flags)"
dep ensure -no-vendor
[ -e vendor/github.com/karlmutch/duat ] || mkdir -p vendor/github.com/karlmutch/duat
mkdir -p cmd/semver/bin
CGO_ENABLED=0 go build -ldflags "$flags" -o cmd/semver/bin/semver cmd/semver/*.go
cp cmd/semver/bin/semver $GOPATH/bin/.
go build -ldflags "$flags" -race -o cmd/semver/bin/semver-race cmd/semver/*.go
CGO_ENABLED=0 go test -ldflags "$flags" -coverpkg="./..." -c -o cmd/semver/bin/semver-run-coverage cmd/semver/*.go
CGO_ENABLED=0 go test -ldflags "$flags" -coverpkg="./..." -c -o cmd/semver/bin/semver-test-coverage cmd/semver/*.go
go test -ldflags "$flags" -race -c -o cmd/semver/bin/semver-test cmd/semver/*.go
if [ -z "$PRE_RELEASE" ]; then
    if ! [ -z "${SEMVER}" ]; then
        if ! [ -z "${GITHUB_TOKEN}" ]; then
            github-release release --user karlmutch --repo duat --tag ${SEMVER} --pre-release && \
            github-release upload --user karlmutch --repo duat  --tag ${SEMVER} --name semver --file cmd/semver/bin/semver
        fi
    fi
fi
