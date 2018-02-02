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
[ -e vendor/github.com/karlmutch/bump-ver ] || mkdir -p vendor/github.com/karlmutch/bump-ver
mkdir -p cmd/bump-ver/bin
CGO_ENABLED=0 go build -ldflags "-X github.com/karlmutch/bump-ver/version.BuildTime=$DATE -X github.com/karlmutch/bump-ver/version.GitHash=$HASH" -o cmd/bump-ver/bin/bump-ver cmd/bump-ver/*.go
cp cmd/bump-ver/bin/bump-ver $GOPATH/bin/.
go build -ldflags "-X github.com/karlmutch/bump-ver/version.BuildTime=$DATE -X github.com/karlmutch/bump-ver/version.GitHash=$HASH" -race -o cmd/bump-ver/bin/bump-ver-race cmd/bump-ver/*.go
CGO_ENABLED=0 go test -ldflags "-X github.com/karlmutch/bump-ver/version.TestRunMain=Use -X github.com/karlmutch/bump-ver/version.BuildTime=$DATE -X github.com/karlmutch/bump-ver/version.GitHash=$HASH" -coverpkg="./..." -c -o cmd/bump-ver/bin/bump-ver-run-coverage cmd/bump-ver/*.go
CGO_ENABLED=0 go test -ldflags "-X github.com/karlmutch/bump-ver/version.BuildTime=$DATE -X github.com/karlmutch/bump-ver/version.GitHash=$HASH" -coverpkg="./..." -c -o cmd/bump-ver/bin/bump-ver-test-coverage cmd/bump-ver/*.go
go test -ldflags "-X github.com/karlmutch/bump-ver/version.BuildTime=$DATE -X github.com/karlmutch/bump-ver/version.GitHash=$HASH" -race -c -o cmd/bump-ver/bin/bump-ver-test cmd/bump-ver/*.go
if ! [ -z "${GITHUB_TOKEN}" ]; then
    github-release release --user karlmutch --repo bump-ver --tag ${TRAVIS_TAG} --pre-release && \
    github-release upload --user karlmutch --repo bump-ver  --tag ${TRAVIS_TAG} --name bump-ver --file cmd/bump-ver/bin/bump-ver
fi
