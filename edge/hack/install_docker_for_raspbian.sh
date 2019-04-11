#!/bin/bash
apt-get update
apt-get install -y apt-transport-https  ca-certificates curl gnupg2 software-properties-common
curl -fsSL https://download.docker.com/linux/raspbian/gpg | apt-key add -
echo "deb [arch=armhf]  https://download.docker.com/linux/raspbian stretch stable" | tee /etc/apt/sources.list.d/docker.list 
apt-get update && apt-get install -y docker-ce docker-ce-cli containerd.io