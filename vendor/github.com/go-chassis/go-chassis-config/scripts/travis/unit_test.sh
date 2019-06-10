#!/bin/sh
set -e

go get -d -u github.com/stretchr/testify/assert

mkdir -p $GOPATH/src/github.com/fsnotify
mkdir -p $GOPATH/src/github.com/spf13
mkdir -p $GOPATH/src/github.com/stretchr
mkdir -p $GOPATH/src/golang.org/x
mkdir -p $GOPATH/src/github.com/cenkalti

go get gopkg.in/yaml.v2

cd $GOPATH/src/github.com/go-chassis
git clone https://github.com/go-chassis/go-chassis.git
git clone https://github.com/go-chassis/http-client.git
git clone https://github.com/go-chassis/paas-lager.git
git clone https://github.com/go-chassis/auth.git
git clone https://github.com/go-chassis/go-archaius.git

cd $GOPATH/src/github.com/fsnotify
git clone https://github.com/fsnotify/fsnotify.git
cd fsnotify
git reset --hard 629574ca2a5df945712d3079857300b5e4da0236

cd $GOPATH/src/github.com/spf13
git clone https://github.com/spf13/cast.git
cd cast
git reset --hard acbeb36b902d72a7a4c18e8f3241075e7ab763e4

cd $GOPATH/src/github.com/stretchr
rm -rf testify
git clone https://github.com/stretchr/testify.git
cd testify
git reset --hard 87b1dfb5b2fa649f52695dd9eae19abe404a4308

cd $GOPATH/src/golang.org/x
git clone https://github.com/golang/sys.git
git clone https://github.com/golang/net.git

cd $GOPATH/src/github.com/cenkalti
git clone https://github.com/cenkalti/backoff.git
cd backoff
git reset --hard 3db60c813733fce657c114634171689bbf1f8dee

cd $GOPATH/src/github.com/go-chassis/go-cc-client
#Start unit test
for d in $(go list ./...); do
    echo $d
    echo $GOPATH
    cd $GOPATH/src/$d
    if [ $(ls | grep _test.go | wc -l) -gt 0 ]; then
        go test 
    fi
done


