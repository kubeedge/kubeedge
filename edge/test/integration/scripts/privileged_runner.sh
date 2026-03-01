#!/usr/bin/env bash

# Copyright 2026 The KubeEdge Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

function configure_privileged_runner() {
  local context="${1:-integration tests}"

  SUDO=()
  if [[ "$(id -u)" -eq 0 ]]; then
    return 0
  fi

  if ! command -v sudo >/dev/null 2>&1; then
    echo "sudo is required for ${context} when not running as root"
    exit 1
  fi

  if [[ -t 0 ]]; then
    SUDO=(sudo)
    return 0
  fi

  if sudo -n true >/dev/null 2>&1; then
    SUDO=(sudo -n)
    return 0
  fi

  echo "passwordless sudo is required for ${context} in non-interactive mode"
  exit 1
}
