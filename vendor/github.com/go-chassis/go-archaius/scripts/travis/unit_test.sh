#!/bin/sh
set -e

cd $GOPATH/src/github.com/go-chassis/go-archaius
#Start unit test
for d in $(go list ./... | grep -v configcenter-source); do
    echo $d
    echo $GOPATH
    cd $GOPATH/src/$d
    if [ $(ls | grep _test.go | wc -l) -gt 0 ]; then
        go test 
    fi
done
