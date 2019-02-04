FROM golang:1.11

LABEL "name"="stencil"
LABEL "version"="development"

LABEL "com.github.actions.name"="stencil"
LABEL "com.github.actions.description"="Run duat stencil on a templated file using the Go template norms, https://golang.org/pkg/text/template/"
LABEL "com.github.actions.icon"="terminal"
LABEL "com.github.actions.color"="blue"

LABEL "repository"="https://github.com/karlmutch/duat"
LABEL "homepage"="https://github.com/karlmutch/duat/actions"
LABEL "maintainer"="karl Mutch <karlmutch@gmail.com>"

RUN apt-get update && \
    apt-get install -y jq && \
    apt-get clean && \
    go get -u github.com/karlmutch/duat/cmd/stencil && \
    go install github.com/karlmutch/duat/cmd/stencil

ADD entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
