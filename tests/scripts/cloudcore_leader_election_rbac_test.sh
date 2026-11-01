#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

assert_lease_verbs() {
  local file="$1"
  local line
  line="$(awk '
    /apiGroups:/ { in_coordination = ($0 ~ /"coordination\.k8s\.io"/) ? 1 : 0; in_leases = 0; next }
    in_coordination && /resources:/ { in_leases = ($0 ~ /"leases"/) ? 1 : 0; next }
    in_coordination && in_leases && /verbs:/ { print; exit }
  ' "${ROOT_DIR}/${file}")"

  if [[ -z "${line}" ]]; then
    echo "missing leases verbs in ${file}" >&2
    exit 1
  fi

  for verb in get list watch create update patch delete; do
    if [[ "${line}" != *"\"${verb}\""* ]]; then
      echo "missing lease verb ${verb} in ${file}: ${line}" >&2
      exit 1
    fi
  done
}

assert_lease_verbs "manifests/charts/cloudcore/templates/rbac_cloudcore.yaml"
assert_lease_verbs "build/cloud/03-clusterrole.yaml"
assert_lease_verbs "build/cloud/ha/01-ha-prepare.yaml"
