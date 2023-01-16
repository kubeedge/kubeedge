# Copyright 2022 The KubeEdge Authors.
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

FROM golang:1.17.13-alpine3.16 AS builder

ARG GO_LDFLAGS

RUN go install github.com/onsi/ginkgo/ginkgo@v1.16.5

COPY . /go/src/github.com/kubeedge/kubeedge

RUN cp /go/src/github.com/kubeedge/kubeedge/build/conformance/kubernetes/kube_conformance_test.go \
    /go/src/github.com/kubeedge/kubeedge/tests/e2e/

RUN cd /go/src/github.com/kubeedge/kubeedge && go mod vendor

RUN CGO_ENABLED=0 GO111MODULE=off ginkgo build -ldflags "-w -s -extldflags -static" -r /go/src/github.com/kubeedge/kubeedge/tests/e2e

RUN CGO_ENABLED=0 GO111MODULE=off go build -v -o /usr/local/bin/e2e-runner -ldflags "$GO_LDFLAGS -w -s" \
   /go/src/github.com/kubeedge/kubeedge/build/conformance/e2e-runner

FROM alpine:3.13

COPY --from=builder /go/bin/ginkgo /usr/local/bin/ginkgo

COPY --from=builder /usr/local/bin/e2e-runner /usr/local/bin/e2e-runner

COPY --from=builder /go/src/github.com/kubeedge/kubeedge/tests/e2e/e2e.test /usr/local/bin/e2e.test

COPY --from=builder  /go/src/github.com/kubeedge/kubeedge/build/conformance/kubernetes/edge_skip_case.yaml /testdata/edge_skip_case.yaml

RUN mkdir -p /tmp/results

ENTRYPOINT ["e2e-runner"]