#!/bin/bash

if ( find /project -maxdepth 0 -empty | read v );
then
  echo "source code must be mounted into the /project directory"
  exit 990
fi

set -e
set -o pipefail

[ -e vendor/github.com/karlmutch/duat ] || mkdir -p vendor/github.com/karlmutch/duat

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
mkdir -p examples/artifact/bin
CGO_ENABLED=0 go build -ldflags "$flags" -o example/artifact/bin/artifact example/artifact/*.go
cp example/artifact/bin/artifact $GOPATH/bin/.
