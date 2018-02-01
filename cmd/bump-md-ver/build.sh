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
dep ensure -no-vendor
[ -e vendor/github.com/karlmutch/bump-md-ver ] || mkdir -p vendor/github.com/karlmutch/bump-md-ver
mkdir -p cmd/bump-md-ver/bin
CGO_ENABLED=0 go build -ldflags "-X github.com/karlmutch/bump-md-ver/version.BuildTime=$DATE -X github.com/karlmutch/bump-md-ver/version.GitHash=$HASH" -o cmd/bump-md-ver/bin/bump-md-ver cmd/bump-md-ver/*.go
go build -ldflags "-X github.com/karlmutch/bump-md-ver/version.BuildTime=$DATE -X github.com/karlmutch/bump-md-ver/version.GitHash=$HASH" -race -o cmd/bump-md-ver/bin/bump-md-ver-race cmd/bump-md-ver/*.go
CGO_ENABLED=0 go test -ldflags "-X github.com/karlmutch/bump-md-ver/version.TestRunMain=Use -X github.com/karlmutch/bump-md-ver/version.BuildTime=$DATE -X github.com/karlmutch/bump-md-ver/version.GitHash=$HASH" -coverpkg="./..." -c -o cmd/bump-md-ver/bin/bump-md-ver-run-coverage cmd/bump-md-ver/*.go
CGO_ENABLED=0 go test -ldflags "-X github.com/karlmutch/bump-md-ver/version.BuildTime=$DATE -X github.com/karlmutch/bump-md-ver/version.GitHash=$HASH" -coverpkg="./..." -c -o cmd/bump-md-ver/bin/bump-md-ver-test-coverage cmd/bump-md-ver/*.go
go test -ldflags "-X github.com/karlmutch/bump-md-ver/version.BuildTime=$DATE -X github.com/karlmutch/bump-md-ver/version.GitHash=$HASH" -race -c -o cmd/bump-md-ver/bin/bump-md-ver-test cmd/bump-md-ver/*.go
if ! [ -z "${GITHUB_TOKEN}" ]; then
    github-release release --user karlmutch --repo bump-md-ver --tag ${TRAVIS_TAG} --pre-release && \
    github-release upload --user karlmutch --repo bump-md-ver  --tag ${TRAVIS_TAG} --name bump-md-ver --file cmd/bump-md-ver/bin/bump-md-v
fi
