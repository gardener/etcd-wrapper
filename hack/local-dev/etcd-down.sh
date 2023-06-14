#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

TARGET_NAMESPACE="default"
USAGE=""

function create_usage() {
  usage=$(printf '%s\n' "
  usage: $(basename $0) [Options]
  Options:
   -n | --namespace <namespace> kubernetes namespace where etcd resources are created. If not specified uses 'default'
   ")
  echo "${usage}"
}

function check_prerequisites() {
  if ! command -v skaffold &>/dev/null; then
    echo -e "skaffold is not installed. Please install, refer: https://skaffold.dev/docs/install/"
    exit 1
  fi
  if ! command -v kubectl &>/dev/null; then
    echo -e "kubectl is not installed. Please install, refer: https://kubernetes.io/docs/tasks/tools/"
    exit 1
  fi
}

function parse_flags() {
  while test $# -gt 0; do
    case "$1" in
    --namespace | -n)
      shift
      TARGET_NAMESPACE="$1"
      ;;
    --help | -h)
      shift
      echo "${USAGE}"
      exit 0
      ;;
    esac
    shift
  done
}

function delete_pvcs() {
  resp=$(kubectl get pvc -n "${TARGET_NAMESPACE}" -o name)
  if [[ -z "$resp" ]]; then
    echo "no pvcs found in namespace ${TARGET_NAMESPACE}"
    return
  fi
  pvcs=( "${resp}" )
  for pvc in "${pvcs[@]}"; do
    echo "> Deleting pvc ${pvc}..."
    kubectl delete ${pvc} -n "${TARGET_NAMESPACE}"
  done
}

function main() {
  check_prerequisites
  parse_flags "$@"
  echo "> Deleting etcd resources in namespace: ${TARGET_NAMESPACE}..."
  skaffold delete -n "${TARGET_NAMESPACE}"
  echo "> Deleting any existing pvc..."
  delete_pvcs
  echo "> All etcd resources deleted from namespace: ${TARGET_NAMESPACE}"
}

USAGE=$(create_usage)
main "$@"
