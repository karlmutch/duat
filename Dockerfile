FROM ubuntu:16.04

MAINTAINER karlmutch@gmail.com

ENV LANG C.UTF-8

RUN apt-get -y update

RUN \
    apt-get -y install git software-properties-common wget openssl ssh curl jq apt-utils unzip python-pip sudo && \
    apt-get clean && \
    apt-get autoremove && \
    pip install awscli --upgrade

RUN \
    apt-get update

ENV GO_VERSION 1.15.6

RUN cd /home/${USER} && \
    mkdir -p /home/${USER}/go && \
    wget -O /tmp/go.tgz https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar xzf /tmp/go.tgz && \
    rm /tmp/go.tgz
ENV GOPATH=/project

RUN mkdir -p /project/src/github.com/karlmutch/duat && \
    go get github.com/erning/gorun && \
    cp -r /makisu-context/. /project/src/github.com/karlmutch/duat/.

WORKDIR /project/src/github.com/karlmutch/duat

CMD go mod vendor ; go run ./build.go -dirs cmd,example -r
