#!/bin/bash

GO_VERSION=1.15 \
GOOS=linux \
GOARCH=amd64 \
GOROOT=/home/go \
GOPATH=/home/go-tools
PATH=$GOPATH/bin:$GOROOT/bin:$PATH

mkdir -p /home/go && \
mkdir -p /home/go-tools && \
chown -R vagrant:vagrant /home/go && \
chown -R vagrant:vagrant /home/go-tools

curl -fsSL https://storage.googleapis.com/golang/go$GO_VERSION.$GOOS-$GOARCH.tar.gz | tar -C /home -xzv

go get -u -v github.com/mdempsky/gocode && \
go get -u -v github.com/uudashr/gopkgs/cmd/gopkgs && \
go get -u -v github.com/ramya-rao-a/go-outline && \
go get -u -v github.com/acroca/go-symbols && \
go get -u -v golang.org/x/tools/cmd/guru && \
go get -u -v golang.org/x/tools/cmd/gorename && \
go get -u -v github.com/fatih/gomodifytags && \
go get -u -v github.com/haya14busa/goplay/cmd/goplay && \
go get -u -v github.com/josharian/impl && \
go get -u -v github.com/tylerb/gotype-live && \
go get -u -v github.com/rogpeppe/godef && \
go get -u -v github.com/zmb3/gogetdoc && \
go get -u -v golang.org/x/tools/cmd/goimports && \
go get -u -v github.com/sqs/goreturns && \
go get -u -v winterdrache.de/goformat/goformat && \
go get -u -v golang.org/x/lint/golint && \
go get -u -v github.com/cweill/gotests/... && \
go get -u -v github.com/alecthomas/gometalinter && \
go get -u -v honnef.co/go/tools/... && \
GO111MODULE=on go get github.com/golangci/golangci-lint/cmd/golangci-lint && \
go get -u -v github.com/mgechev/revive && \
go get -u -v github.com/sourcegraph/go-langserver && \
go get -u -v github.com/go-delve/delve/cmd/dlv && \
go get -u -v github.com/davidrjenni/reftools/cmd/fillstruct && \
go get -u -v github.com/godoctor/godoctor

go get -u -v -d github.com/stamblerre/gocode && \
go build -o $GOPATH/bin/gocode-gomod github.com/stamblerre/gocode
