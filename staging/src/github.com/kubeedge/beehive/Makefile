#
# Copyright 2019 The KubeEdge Authors.
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
#


PKG=github.com/kubeedge/beehive

# Project packages.
PACKAGES=$(shell go list ./... | grep -v "/vendor/")
INTEGRATION_PACKAGE=${PKG}

#Replaces ":" (*nix), ";" (windows) with newline for easy parsing
GOPATHS=$(shell echo ${GOPATH} | tr ":" "\n" | tr ";" "\n")

# Flags passed to `go test`
TESTFLAGS ?= -v

.PHONY: all test lint depcheck benchmark coverage

.DEFAULT: default

all: test lint depcheck ## default

depcheck: ## Check if imports, Gopkg.toml, and Gopkg.lock are in sync
	dep check

lint: ## golangci-lint check
	golangci-lint run --disable-all -E gofmt -E golint ./...
	go vet ./...

test: ## test case
	@go test ${TESTFLAGS} $(filter-out ${INTEGRATION_PACKAGE},${PACKAGES})

# https://deepzz.com/post/study-golang-test.html
# https://deepzz.com/post/the-command-flag-of-go-test.html
benchmark: ## run benchmarks tests
	@go test ${TESTFLAGS} $(filter-out ${INTEGRATION_PACKAGE},${PACKAGES})  -bench . -run Benchmark

coverage: ## generate coverprofiles from the unit tests, except tests that require root
	@rm -f coverage.txt
	@go test -i ${TESTFLAGS} $(filter-out ${INTEGRATION_PACKAGE},${PACKAGES}) 2> /dev/null
	@( for pkg in $(filter-out ${INTEGRATION_PACKAGE},${PACKAGES}); do \
		go test ${TESTFLAGS} \
			-cover \
			-coverprofile=profile.out \
			-covermode=atomic $$pkg || exit; \
		if [ -f profile.out ]; then \
			cat profile.out >> coverage.txt; \
			$(RM) profile.out; \
		fi; \
	done )
