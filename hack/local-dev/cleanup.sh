#!/usr/bin/env bash
# Copyright 2023 SAP SE or an SAP affiliate company
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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
PROJECT_DIR="$(cd "$(dirname "$SCRIPT_DIR")/.." &>/dev/null && pwd)"

function cleanup_resources() {
  echo "> Cleaning up all generated files..."
  rm -rf "${SCRIPT_DIR}"/manifests/common
  rm -rf "${SCRIPT_DIR}"/manifests/singlenode
  rm -rf "${SCRIPT_DIR}"/manifests/multinode
  rm -rf "${SCRIPT_DIR}"/secrets

  etcd_config_path="${SCRIPT_DIR}"/config/etcd.config.yaml
  if [ -f "${etcd_config_path}" ]; then
    rm "${etcd_config_path}"
  fi
  kind_config_path="${SCRIPT_DIR}"/config/kind.config.yaml
  if [ -f "${kind_config_path}" ]; then
    rm "${kind_config_path}"
  fi
  skaffold_path="${PROJECT_DIR}"/skaffold.yaml
  if [ -f "${skaffold_path}" ]; then
    rm "${skaffold_path}"
  fi
}
cat <<EOF
📌 NOTE:
  Cleanup will remove all resources including skaffold YAML file.
  skaffold YAML is used to cleanup all etcd resources in etcd-down.sh script.
  Cleaning up generated files will hamper usage of etcd-down.sh script.
  Do you wish to continue?
EOF
select yn in "Yes" "No"; do
  case $yn in
  Yes)
    cleanup_resources
    break
    ;;
  No) exit ;;
  esac
done
