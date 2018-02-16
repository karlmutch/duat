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
flags='-X github.com/karlmutch/bump-ver/version.BuildTime=$DATE -X github.com/karlmutch/bump-ver/version.GitHash=$HASH -X github.com/karlmutch/bump-ver/version.SemVer="${SEMVER}"'
flags="$(eval echo $flags)"
dep ensure -no-vendor
[ -e vendor/github.com/karlmutch/bump-ver ] || mkdir -p vendor/github.com/karlmutch/bump-ver
mkdir -p cmd/bump-ver/bin
CGO_ENABLED=0 go build -ldflags "$flags" -o cmd/bump-ver/bin/bump-ver cmd/bump-ver/*.go
cp cmd/bump-ver/bin/bump-ver $GOPATH/bin/.
go build -ldflags "$flags" -race -o cmd/bump-ver/bin/bump-ver-race cmd/bump-ver/*.go
CGO_ENABLED=0 go test -ldflags "$flags" -coverpkg="./..." -c -o cmd/bump-ver/bin/bump-ver-run-coverage cmd/bump-ver/*.go
CGO_ENABLED=0 go test -ldflags "$flags" -coverpkg="./..." -c -o cmd/bump-ver/bin/bump-ver-test-coverage cmd/bump-ver/*.go
go test -ldflags "$flags" -race -c -o cmd/bump-ver/bin/bump-ver-test cmd/bump-ver/*.go
if [ -z "$PRE_RELEASE" ]; then
    if ! [ -z "${SEMVER}" ]; then
        if ! [ -z "${GITHUB_TOKEN}" ]; then
            github-release release --user karlmutch --repo bump-ver --tag ${SEMVER} --pre-release && \
            github-release upload --user karlmutch --repo bump-ver  --tag ${SEMVER} --name bump-ver --file cmd/bump-ver/bin/bump-ver
        fi
    fi
fi
