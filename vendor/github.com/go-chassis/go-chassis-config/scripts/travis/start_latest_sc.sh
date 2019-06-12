#!/usr/bin/env bash
cd ../..
kill -9 $(ps aux | grep etcd | grep -v 'grep' | awk '{print $2}')
kill -9 $(ps aux | grep service-center | grep -v 'grep' | awk '{print $2}')
rm -rf etcd apache
mkdir -p etcd
cd etcd
wget --no-check-certificate https://github.com/coreos/etcd/releases/download/v3.1.8/etcd-v3.1.8-linux-amd64.tar.gz
tar -xvf etcd-v3.1.8-linux-amd64.tar.gz
./etcd-v3.1.8-linux-amd64/etcd > start-etcd.log 2>&1 &
cd -
mkdir -p apache
cd apache
git clone https://github.com/apache/servicecomb-service-center.git
cd servicecomb-service-center
gvt restore
go build -o servicecomb-service-center
cp -r etc/conf .
./servicecomb-service-center > start-sc.log 2>&1 &
