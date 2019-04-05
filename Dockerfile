FROM ubuntu:16.04

MAINTAINER karlmutch@gmail.com

ENV LANG C.UTF-8

RUN apt-get -y update

RUN apt-get -y install git software-properties-common wget openssl ssh curl jq apt-utils unzip python-pip sudo && \
    apt-get clean && \
    apt-get autoremove && \
    pip install awscli --upgrade

RUN add-apt-repository ppa:gophers/archive && \
    apt-get update && \
    apt-get -y install golang-1.11-go

ENV GOPATH=/project
ENV GOROOT=/usr/lib/go-1.11
ENV PATH=$PATH:/usr/lib/go-1.11/bin:$GOPATH/bin

RUN mkdir -p /project/src/github.com/karlmutch/duat && \
    go get -u github.com/golang/dep/cmd/dep && \
    go get github.com/erning/gorun && \
    cp -r /makisu-context/. /project/src/github.com/karlmutch/duat/.

WORKDIR /project/src/github.com/karlmutch/duat

CMD go run ./build.go -dirs cmd,example -r
