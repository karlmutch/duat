FROM golang:1.10rc1

MAINTAINER karlmutch@gmail.com

ENV LANG C.UTF-8

ARG USER
ENV USER ${USER}
ARG USER_ID
ENV USER_ID ${USER_ID}
ARG USER_GROUP_ID
ENV USER_GROUP_ID ${USER_GROUP_ID}

RUN apt-get -y update

RUN apt-get -y install git software-properties-common wget openssl ssh curl jq apt-utils unzip python-pip && \
    apt-get clean && \
    apt-get autoremove && \
    pip install awscli --upgrade && \
    groupadd -f -g ${USER_GROUP_ID} ${USER} && \
    useradd -g ${USER_GROUP_ID} -u ${USER_ID} -ms /bin/bash ${USER}

USER ${USER}
WORKDIR /home/${USER}

ENV GOPATH=/project
VOLUME /project
WORKDIR /project/src/github.com/karlmutch/bump-md-ver

CMD /bin/bash -C ./all-build.sh
