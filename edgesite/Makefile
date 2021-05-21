# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARCH ?= amd64
ALL_ARCH = amd64 arm arm64 ppc64le s390x

GOPATH ?= $(GOPATH)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
INSTALL_LOCATION:=$(shell go env GOPATH)/bin
GOLANGCI_LINT_VERSION ?= 1.35.2
GOSEC_VERSION ?= 2.5.0

REGISTRY ?= gcr.io/$(shell gcloud config get-value project)
STAGING_REGISTRY := gcr.io/k8s-staging-kas-network-proxy

SERVER_IMAGE_NAME ?= proxy-server
AGENT_IMAGE_NAME ?= proxy-agent
TEST_CLIENT_IMAGE_NAME ?= proxy-test-client
TEST_SERVER_IMAGE_NAME ?= http-test-server

SERVER_FULL_IMAGE ?= $(REGISTRY)/$(SERVER_IMAGE_NAME)
AGENT_FULL_IMAGE ?= $(REGISTRY)/$(AGENT_IMAGE_NAME)
TEST_CLIENT_FULL_IMAGE ?= $(REGISTRY)/$(TEST_CLIENT_IMAGE_NAME)
TEST_SERVER_FULL_IMAGE ?= $(REGISTRY)/$(TEST_SERVER_IMAGE_NAME)

TAG ?= $(shell git rev-parse HEAD)

DOCKER_CMD ?= docker
DOCKER_CLI_EXPERIMENTAL ?= enabled
PROXY_SERVER_IP ?= 127.0.0.1
## --------------------------------------
## Testing
## --------------------------------------
mock_gen:
	mkdir -p proto/agent/mocks
	mockgen sigs.k8s.io/apiserver-network-proxy/proto/agent AgentService_ConnectServer > proto/agent/mocks/agent_mock.go
	cat hack/go-license-header.txt proto/agent/mocks/agent_mock.go > proto/agent/mocks/agent_mock.licensed.go
	mv proto/agent/mocks/agent_mock.licensed.go proto/agent/mocks/agent_mock.go

.PHONY: test
test:
	GO111MODULE=on go test -race sigs.k8s.io/apiserver-network-proxy/...

## --------------------------------------
## Binaries
## --------------------------------------

bin:
	mkdir -p bin

.PHONY: build
build: bin/proxy-agent bin/proxy-server

bin/proxy-agent: proto/agent/agent.pb.go konnectivity-client/proto/client/client.pb.go bin cmd/agent/main.go
	GO111MODULE=on go build -o bin/proxy-agent cmd/agent/main.go

bin/proxy-server: proto/agent/agent.pb.go konnectivity-client/proto/client/client.pb.go bin cmd/server/main.go
	GO111MODULE=on go build -o bin/proxy-server cmd/server/main.go

## --------------------------------------
## Linting
## --------------------------------------

.PHONY: lint
lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(INSTALL_LOCATION) v$(GOLANGCI_LINT_VERSION)
	$(INSTALL_LOCATION)/golangci-lint run --no-config --disable-all --enable=gofmt,golint,gosec --fix --verbose --timeout 3m

## --------------------------------------
## Go
## --------------------------------------

.PHONY: mod-download
mod-download:
	go mod download

## --------------------------------------
## Proto
## --------------------------------------

.PHONY: gen
gen: mod-download proto/agent/agent.pb.go konnectivity-client/proto/client/client.pb.go mock_gen

konnectivity-client/proto/client/client.pb.go: konnectivity-client/proto/client/client.proto
	mkdir -p ${GOPATH}/src
	protoc -I . konnectivity-client/proto/client/client.proto --go_out=plugins=grpc:${GOPATH}/src
	cat hack/go-license-header.txt ${GOPATH}/src/sigs.k8s.io/apiserver-network-proxy/konnectivity-client/proto/client/client.pb.go > konnectivity-client/proto/client/client.licensed.go
	mv konnectivity-client/proto/client/client.licensed.go konnectivity-client/proto/client/client.pb.go

proto/agent/agent.pb.go: proto/agent/agent.proto
	mkdir -p ${GOPATH}/src
	protoc -I . proto/agent/agent.proto --go_out=plugins=grpc:${GOPATH}/src
	cat hack/go-license-header.txt ${GOPATH}/src/sigs.k8s.io/apiserver-network-proxy/proto/agent/agent.pb.go > proto/agent/agent.licensed.go
	mv proto/agent/agent.licensed.go proto/agent/agent.pb.go

## --------------------------------------
## Certs
## --------------------------------------

easy-rsa.tar.gz:
	curl -L -o easy-rsa.tar.gz --connect-timeout 20 --retry 6 --retry-delay 2 https://github.com/OpenVPN/easy-rsa/archive/refs/tags/v3.0.8.tar.gz

easy-rsa-master: easy-rsa.tar.gz
	tar xvf easy-rsa.tar.gz
	mv easy-rsa-3.0.8 easy-rsa-master

cfssl:
	@if ! command -v cfssl &> /dev/null; then \
		curl --retry 10 -L -o cfssl https://github.com/cloudflare/cfssl/releases/download/v1.5.0/cfssl_1.5.0_$(GOOS)_$(GOARCH); \
		chmod +x cfssl; \
	fi

cfssljson:
	@if ! command -v cfssljson &> /dev/null; then \
		curl --retry 10 -L -o cfssljson https://github.com/cloudflare/cfssl/releases/download/v1.5.0/cfssljson_1.5.0_$(GOOS)_$(GOARCH); \
		chmod +x cfssljson; \
	fi

.PHONY: certs
certs: export PATH := $(shell pwd):$(PATH)
certs: easy-rsa-master cfssl cfssljson
	# set up easy-rsa
	cp -rf easy-rsa-master/easyrsa3 easy-rsa-master/master
	cp -rf easy-rsa-master/easyrsa3 easy-rsa-master/agent
	# create the client <-> server-proxy connection certs
	cd easy-rsa-master/master; \
	./easyrsa init-pki; \
	./easyrsa --batch "--req-cn=127.0.0.1@$(date +%s)" build-ca nopass; \
	./easyrsa --subject-alt-name="DNS:kubernetes,DNS:localhost,IP:127.0.0.1" build-server-full "proxy-master" nopass; \
	./easyrsa build-client-full proxy-client nopass; \
	echo '{"signing":{"default":{"expiry":"43800h","usages":["signing","key encipherment","client auth"]}}}' > "ca-config.json"; \
	echo '{"CN":"proxy","names":[{"O":"system:nodes"}],"hosts":[""],"key":{"algo":"rsa","size":2048}}' | cfssl gencert -ca=pki/ca.crt -ca-key=pki/private/ca.key -config=ca-config.json - | cfssljson -bare proxy
	mkdir -p certs/master
	cp -r easy-rsa-master/master/pki/private certs/master
	cp -r easy-rsa-master/master/pki/issued certs/master
	cp easy-rsa-master/master/pki/ca.crt certs/master/issued
	# create the agent <-> server-proxy connection certs
	cd easy-rsa-master/agent; \
	./easyrsa init-pki; \
	./easyrsa --batch "--req-cn=$(PROXY_SERVER_IP)@$(date +%s)" build-ca nopass; \
	./easyrsa --subject-alt-name="DNS:kubernetes,DNS:localhost,IP:$(PROXY_SERVER_IP)" build-server-full "proxy-master" nopass; \
	./easyrsa build-client-full proxy-agent nopass; \
	echo '{"signing":{"default":{"expiry":"43800h","usages":["signing","key encipherment","agent auth"]}}}' > "ca-config.json"; \
	echo '{"CN":"proxy","names":[{"O":"system:nodes"}],"hosts":[""],"key":{"algo":"rsa","size":2048}}' | cfssl gencert -ca=pki/ca.crt -ca-key=pki/private/ca.key -config=ca-config.json - | cfssljson -bare proxy
	mkdir -p certs/agent
	cp -r easy-rsa-master/agent/pki/private certs/agent
	cp -r easy-rsa-master/agent/pki/issued certs/agent
	cp easy-rsa-master/agent/pki/ca.crt certs/agent/issued

## --------------------------------------
## Docker
## --------------------------------------

.PHONY: docker-build
docker-build: docker-build/proxy-agent docker-build/proxy-server

.PHONY: docker-push
docker-push: docker-push/proxy-agent docker-push/proxy-server

.PHONY: docker-build/proxy-agent
docker-build/proxy-agent: cmd/agent/main.go proto/agent/agent.pb.go
	@[ "${TAG}" ] || ( echo "TAG is not set"; exit 1 )
	echo "Building proxy-agent for ${ARCH}"
	${DOCKER_CMD} build . --build-arg ARCH=$(ARCH) -f artifacts/images/agent-build.Dockerfile -t ${AGENT_FULL_IMAGE}-$(ARCH):${TAG}

.PHONY: docker-push/proxy-agent
docker-push/proxy-agent: docker-build/proxy-agent
	@[ "${DOCKER_CMD}" ] || ( echo "DOCKER_CMD is not set"; exit 1 )
	${DOCKER_CMD} push ${AGENT_FULL_IMAGE}-$(ARCH):${TAG}

.PHONY: docker-build/proxy-server
docker-build/proxy-server: cmd/server/main.go proto/agent/agent.pb.go
	@[ "${TAG}" ] || ( echo "TAG is not set"; exit 1 )
	echo "Building proxy-server for ${ARCH}"
	${DOCKER_CMD} build . --build-arg ARCH=$(ARCH) -f artifacts/images/server-build.Dockerfile -t ${SERVER_FULL_IMAGE}-$(ARCH):${TAG}

.PHONY: docker-push/proxy-server
docker-push/proxy-server: docker-build/proxy-server
	@[ "${DOCKER_CMD}" ] || ( echo "DOCKER_CMD is not set"; exit 1 )
	${DOCKER_CMD} push ${SERVER_FULL_IMAGE}-$(ARCH):${TAG}

## --------------------------------------
## Docker — All ARCH
## --------------------------------------

.PHONY: docker-build-all
docker-build-all: $(addprefix docker-build/proxy-agent-,$(ALL_ARCH)) $(addprefix docker-build/proxy-server-,$(ALL_ARCH))

.PHONY: docker-push-all
docker-push-all: $(addprefix docker-push/proxy-agent-,$(ALL_ARCH)) $(addprefix docker-push/proxy-server-,$(ALL_ARCH))
	$(MAKE) docker-push-manifest/proxy-agent
	$(MAKE) docker-push-manifest/proxy-server

docker-build/proxy-agent-%:
	$(MAKE) ARCH=$* docker-build/proxy-agent

docker-push/proxy-agent-%:
	$(MAKE) ARCH=$* docker-push/proxy-agent

docker-build/proxy-server-%:
	$(MAKE) ARCH=$* docker-build/proxy-server

docker-push/proxy-server-%:
	$(MAKE) ARCH=$* docker-push/proxy-server


.PHONY: docker-push-manifest/proxy-agent
docker-push-manifest/proxy-agent: ## Push the fat manifest docker image.
	## Minimum docker version 18.06.0 is required for creating and pushing manifest images.
	${DOCKER_CMD} manifest create --amend $(AGENT_FULL_IMAGE):$(TAG) $(shell echo $(ALL_ARCH) | sed -e "s~[^ ]*~$(AGENT_FULL_IMAGE)\-&:$(TAG)~g")
	@for arch in $(ALL_ARCH); do ${DOCKER_CMD} manifest annotate --arch $${arch} ${AGENT_FULL_IMAGE}:${TAG} ${AGENT_FULL_IMAGE}-$${arch}:${TAG}; done
	${DOCKER_CMD} manifest push --purge $(AGENT_FULL_IMAGE):$(TAG)

.PHONY: docker-push-manifest/proxy-server
docker-push-manifest/proxy-server: ## Push the fat manifest docker image.
	## Minimum docker version 18.06.0 is required for creating and pushing manifest images.
	${DOCKER_CMD} manifest create --amend $(SERVER_FULL_IMAGE):$(TAG) $(shell echo $(ALL_ARCH) | sed -e "s~[^ ]*~$(SERVER_FULL_IMAGE)\-&:$(TAG)~g")
	@for arch in $(ALL_ARCH); do ${DOCKER_CMD} manifest annotate --arch $${arch} ${SERVER_FULL_IMAGE}:${TAG} ${SERVER_FULL_IMAGE}-$${arch}:${TAG}; done
	${DOCKER_CMD} manifest push --purge $(SERVER_FULL_IMAGE):$(TAG)

## --------------------------------------
## Release
## --------------------------------------

.PHONY: release-staging
release-staging: ## Builds and push container images to the staging bucket.
	REGISTRY=$(STAGING_REGISTRY) $(MAKE) docker-push-all release-alias-tag

.PHONY: release-alias-tag
release-alias-tag: # Adds the tag to the last build tag. BASE_REF comes from the cloudbuild.yaml
	gcloud container images add-tag $(AGENT_FULL_IMAGE):$(TAG) $(AGENT_FULL_IMAGE):$(BASE_REF)
	gcloud container images add-tag $(SERVER_FULL_IMAGE):$(TAG) $(SERVER_FULL_IMAGE):$(BASE_REF)

## --------------------------------------
## Cleanup / Verification
## --------------------------------------

.PHONY: clean
clean:
	go clean -testcache
	rm -rf proto/agent/agent.pb.go konnectivity-client/proto/client/client.pb.go easy-rsa.tar.gz easy-rsa-master cfssl cfssljson certs bin proto/agent/mocks
