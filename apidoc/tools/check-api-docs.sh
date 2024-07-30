#!/usr/bin/env bash

# Determine the root directory of the project
SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")/..

# Define paths using SCRIPT_ROOT
DIFFROOT_OPENAPI="${SCRIPT_ROOT}/apidoc/generated/openapi"
TMP_DIFFROOT_OPENAPI="${SCRIPT_ROOT}/_tmp/apidoc/generated/openapi"
DIFFROOT_SWAGGER="${SCRIPT_ROOT}/apidoc/generated/swagger"
TMP_DIFFROOT_SWAGGER="${SCRIPT_ROOT}/_tmp/apidoc/generated/swagger"

# Create temporary directories for comparison
mkdir -p "${TMP_DIFFROOT_OPENAPI}"
mkdir -p "${TMP_DIFFROOT_SWAGGER}"

# Copy existing files to temporary directories
cp -a "${DIFFROOT_OPENAPI}"/* "${TMP_DIFFROOT_OPENAPI}"
cp -a "${DIFFROOT_SWAGGER}"/* "${TMP_DIFFROOT_SWAGGER}"

# Generate new OpenAPI and Swagger files
"${SCRIPT_ROOT}/apidoc/tools/generate-openapi.sh"
"${SCRIPT_ROOT}/apidoc/tools/update-swagger-docs.sh"

# Compare generated files to see if they are up-to-date
diff -Naupr "${DIFFROOT_OPENAPI}" "${TMP_DIFFROOT_OPENAPI}" || { echo "OpenAPI files need updating. Run the generation scripts."; exit 1; }
diff -Naupr "${DIFFROOT_SWAGGER}" "${TMP_DIFFROOT_SWAGGER}" || { echo "Swagger files need updating. Run the generation scripts."; exit 1; }

# Clean up
rm -rf "${SCRIPT_ROOT}/_tmp"
