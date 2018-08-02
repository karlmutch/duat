FROM ubuntu:16.04

MAINTAINER karlmutch@gmail.com

ENV LANG C.UTF-8

ARG USER
ENV USER ${USER}
ARG USER_ID
ENV USER_ID ${USER_ID}
ARG USER_GROUP_ID
ENV USER_GROUP_ID ${USER_GROUP_ID}

RUN apt-get -y update

RUN apt-get -y install git software-properties-common wget openssl ssh curl jq apt-utils unzip python-pip sudo && \
    apt-get clean && \
    apt-get autoremove && \
    pip install awscli --upgrade && \
    groupadd -f -g ${USER_GROUP_ID} ${USER} && \
    useradd -g ${USER_GROUP_ID} -u ${USER_ID} -ms /bin/bash ${USER}

ENV GO_VERSION 1.10.3

RUN cd /home/${USER} && \
    mkdir -p /home/${USER}/go && \
    wget -O /tmp/go.tgz https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar xzf /tmp/go.tgz && \
    rm /tmp/go.tgz

RUN export IMG_SHA256="d8495994d46ee40180fbd3d3f13f12c81352b08af32cd2a3361db3f1d5503fa2" \
    && curl -sfSL "https://github.com/genuinetools/img/releases/download/v0.4.8/img-linux-amd64" -o "/usr/local/bin/img" \
    && echo "${IMG_SHA256}  /usr/local/bin/img" | sha256sum -c - \
    && chmod a+x "/usr/local/bin/img" \
    && echo "kernel.unprivileged_userns_clone=1" > /etc/sysctl.d/10-userns.conf

ADD assets/runc /usr/sbin/runc
RUN chmod a+x /usr/sbin/runc

ENV PATH=$PATH:/home/${USER}/go/bin
ENV GOROOT=/home/${USER}/go
ENV GOPATH=/project

RUN go get github.com/erning/gorun && \
    mv $GOPATH/bin/gorun /usr/local/bin

#    mount binfmt_misc -t binfmt_misc /proc/sys/fs/binfmt_misc && \
#    echo ':golang:E::go::/usr/local/bin/gorun:OC' >> /proc/sys/fs/binfmt_misc/register

USER ${USER}
VOLUME /project
WORKDIR /project/src/github.com/karlmutch/duat

CMD go run ./build.go -dirs cmd,example -r
