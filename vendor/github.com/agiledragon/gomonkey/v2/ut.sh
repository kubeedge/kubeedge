#!/usr/bin/env bash

set -e
echo "" > coverage.txt

for d in $(go list ./test/...  | grep -v test/fake); do
    echo "--------Run test package: $d"
    GO111MODULE=on go test -gcflags="all=-N -l" -v -coverprofile=profile.out -coverpkg=./... -covermode=atomic $d
    echo "--------Finish test package: $d"
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
