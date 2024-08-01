#!/usr/bin/env bash

# Copyright 2020 The KUBEEDGE_ROOT Authors.
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
set -o errexit
set -o nounset
set -o pipefail


# Determine the root directory of the project
KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

# Define paths using KUBEEDGE_ROOT
DIFFROOT_OPENAPI="${KUBEEDGE_ROOT}/apidoc/generated/openapi"
TMP_DIFFROOT_OPENAPI="${KUBEEDGE_ROOT}/_tmp/apidoc/generated/openapi"
DIFFROOT_SWAGGER="${KUBEEDGE_ROOT}/apidoc/generated/swagger"
TMP_DIFFROOT_SWAGGER="${KUBEEDGE_ROOT}/_tmp/apidoc/generated/swagger"

# Create temporary directories for comparison
mkdir -p "${TMP_DIFFROOT_OPENAPI}"
mkdir -p "${TMP_DIFFROOT_SWAGGER}"

# Copy existing files to temporary directories
cp -a "${DIFFROOT_OPENAPI}"/* "${TMP_DIFFROOT_OPENAPI}"
cp -a "${DIFFROOT_SWAGGER}"/* "${TMP_DIFFROOT_SWAGGER}"

# Generate new OpenAPI and Swagger files
"${KUBEEDGE_ROOT}/apidoc/tools/generate-openapi.sh"
"${KUBEEDGE_ROOT}/apidoc/tools/update-swagger-docs.sh"

# Compare generated files to see if they are up-to-date
diff -Naupr "${DIFFROOT_OPENAPI}" "${TMP_DIFFROOT_OPENAPI}" || { echo "OpenAPI files need updating. Run the generation scripts."; exit 1; }
diff -Naupr "${DIFFROOT_SWAGGER}" "${TMP_DIFFROOT_SWAGGER}" || { echo "Swagger files need updating. Run the generation scripts."; exit 1; }

# Clean up
rm -rf "${KUBEEDGE_ROOT}/_tmp"
