#!/usr/bin/env bash

# get gometalinter(https://github.com/alecthomas/gometalinter)
curl -L https://git.io/vp6lP | sh

export PATH=${PATH}:${GOPATH}/src/kubeedge/bin

gometalinter --disable-all --enable=gofmt --enable=misspell --exclude=vendor ./...
if [ $? != 0 ]; then
        echo "Please fix the warnings!"
	echo "Run hack/update-gofmt.sh if any warnings in gofmt"
        exit 1
fi


