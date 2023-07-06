#!/usr/bin/env bash

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

echo "Cleanup will remove all resources including skaffold YAML file. You will not be able to cleanup etcd resources using etcd-down.sh script. Do you wish to continue?"
select yn in "Yes" "No"; do
  case $yn in
  Yes)
    cleanup_resources
    break
    ;;
  No) exit ;;
  esac
done

