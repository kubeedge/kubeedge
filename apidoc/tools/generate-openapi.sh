#!/usr/bin/env bash

# Copyright 2024 The KubeEdge Authors.

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

set -o errexit
set -o nounset
set -o pipefail

KUBEEDGE_ROOT=$(unset CDPATH && cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)

GOFILE="${KUBEEDGE_ROOT}/vendor/k8s.io/kube-openapi/cmd/openapi-gen/openapi-gen.go"
if ! test -f "${GOFILE}" ; then
    echo "Error: Go file not found." && exit 1
fi
OUTPUT_FILE="${KUBEEDGE_ROOT}/vendor/k8s.io/kube-openapi/cmd/openapi-gen"
go build -o "${OUTPUT_FILE}" "${GOFILE}"

echo "Generating with openapi-gen"
${OUTPUT_FILE}/openapi-gen\
    --input-dirs "github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1" \
    --input-dirs "github.com/kubeedge/kubeedge/pkg/apis/policy/v1alpha1" \
    --input-dirs "github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2" \
    --input-dirs "github.com/kubeedge/kubeedge/pkg/apis/devices/v1beta1" \
    --input-dirs "github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1" \
    --input-dirs "github.com/kubeedge/kubeedge/pkg/apis/reliablesyncs/v1alpha1" \
    --input-dirs "github.com/kubeedge/kubeedge/pkg/apis/rules/v1" \
    --input-dirs "k8s.io/apimachinery/pkg/apis/meta/v1" \
    --input-dirs "k8s.io/api/rbac/v1" \
    --input-dirs "k8s.io/api/core/v1" \
    --input-dirs "k8s.io/apimachinery/pkg/runtime" \
    --input-dirs "k8s.io/apiextensions-apiserver/pkg/apis" \
    --input-dirs "k8s.io/kubernetes/pkg/apis" \
    --input-dirs "k8s.io/apimachinery/pkg/version" \
    --input-dirs "k8s.io/apimachinery/pkg/api/resource" \
    --output-base "${KUBEEDGE_ROOT}/apidoc/generated"\
    --output-package "openapi" \
    --go-header-file "${KUBEEDGE_ROOT}/hack/boilerplate/boilerplate.txt" \
    --output-file-base "zz_generated.openapi"\
	  --v "9"