#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# TARGET_NAMESPACE is the target namespace where the resources will be setup. If not specified then it assumes 'default' namespace is the target namespace where the resources will be setup. If not specified then it assumes 'default' namespace.
TARGET_NAMESPACE=""
# TLS_ENABLED possible values 'true' and 'false' (default). If its value is true then TLS resources will be generated and all communication will be TLS enabled.
TLS_ENABLED="false" #
# KIND_CLUSTER_NAME is the name of the kind cluster. If not specified then kind default is used.
KIND_CLUSTER_NAME=""
ETCD_CLUSTER_SIZE=1
ETCD_CLIENT_SVC_NAME=""
ETCD_PEER_SVC_NAME=""
CERT_EXPIRY="12h"
FORCE_CREATE_PKI_RESOURCES="false"
FORCE_CREATE_KIND_CLUSTER="false"

declare -a PKI_RESOURCES

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"

source "${SCRIPT_DIR}"/generate_pki.sh
source "${SCRIPT_DIR}"/generate_k8s_secrets.sh

function create_usage() {
  usage=$(printf '%s\n' '
  usage: $(basename $0) [Options]
  Options:
   -k | --kind-cluster-name           <kind-cluster-name>             (Optional) name of the kind cluster. if not specified, uses default name "kind"
   -n | --namespace                   <namespace>                     (Optional) kubernetes namespace where etcd resources will be created. if not specified uses "default"
   -s | --cluster-size                <size of etcd cluster>          (Optional) size of an etcd cluster. Supported values are 1 or 3. Defaults to 1
   -t | --tls-enabled                 <is-tls-enabled>                (Optional) controls the TLS communication amongst peers and between etcd and its client.Possible values: ["true" | "false"]. Defaults to "false"
   -i | --etcd-client-svc-name        <name of etcd client service>   (Optional) name of the etcd kubernetes client service. (Required) if TLS has been enabled
   -p | --etcd-peer-svc-name          <name of etcd peer service>     (Optional) name of the etcd kubernetes peer service. (Required) if TLS has been enabled
   -e | --cert-expiry                 <certificate expiry>            (Optional) common expiry for all certificates generated. Defaults to "12h"
   -o | --target-pki-dir              <target PKI directory>          (Optional) target directory to put all generated PKI resources (certificates and keys). (Required) if TLS has been enabled
   --force-create-pki-resources                                       (Optional) Defaults to "false" which means that PKI resources will not be created if they all exists. Even if one the resources does not exist PKI resources will be created again. If it true, then it forces re-creation of all PKI resources
   --force-create-kind-cluster                                        (Optional) Defaults to "false" which means it will not re-create KIND cluster if one with the given name already exists. To force recreation set this flag to "true".
   ')
  echo "${usage}"
}

function initialize_pki_resources_arr() {
  PKI_RESOURCES+=("${TARGET_PKI_DIR}"/ca.pem)
  PKI_RESOURCES+=("${TARGET_PKI_DIR}"/ca-key.pem)
  PKI_RESOURCES+=("${TARGET_PKI_DIR}"/peer-ca.pem)
  PKI_RESOURCES+=("${TARGET_PKI_DIR}"/peer-ca-key.pem)
  PKI_RESOURCES+=("${TARGET_PKI_DIR}"/etcd-server.pem)
  PKI_RESOURCES+=("${TARGET_PKI_DIR}"/etcd-server-key.pem)
  PKI_RESOURCES+=("${TARGET_PKI_DIR}"/etcd-peer.pem)
  PKI_RESOURCES+=("${TARGET_PKI_DIR}"/etcd-peer-key.pem)
  PKI_RESOURCES+=("${TARGET_PKI_DIR}"/etcd-client.pem)
  PKI_RESOURCES+=("${TARGET_PKI_DIR}"/etcd-client-key.pem)
}

function check_prerequisites() {
  if ! command -v docker &>/dev/null; then
    echo -e "docker is not installed. Please install docker, refer: https://docs.docker.com/desktop/"
    exit 1
  fi
  if ! command -v kind &>/dev/null; then
    echo -e "kind command is not found. Please install kind, refer: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
    exit 1
  fi
  if ! command -v skaffold &>/dev/null; then
    echo -e "skaffold is not installed. Please install skaffold, refer: https://skaffold.dev/docs/install/"
    exit 1
  fi
}

function parse_flags() {
  while test $# -gt 0; do
    case "$1" in
    --kind-cluster-name | -k)
      shift
      KIND_CLUSTER_NAME="$1"
      ;;
    --namespace | -n)
      shift
      TARGET_NAMESPACE="$1"
      ;;
    --cluster-size | -s)
      shift
      ETCD_CLUSTER_SIZE=$1
      ;;
    --tls-enabled | -t)
      shift
      TLS_ENABLED=$(echo "$1" | awk '{print tolower($0)}')
      ;;
    --etcd-client-svc-name | -i)
      shift
      ETCD_CLIENT_SVC_NAME="$1"
      ;;
    --etcd-peer-svc-name | -p)
      shift
      ETCD_PEER_SVC_NAME="$1"
      ;;
    --cert-expiry | -e)
      shift
      CERT_EXPIRY="$1"
      ;;
    --target_pki_dir | -o)
      shift
      TARGET_PKI_DIR="$1"
      ;;
    --force-create-pki-resources)
      FORCE_CREATE_PKI_RESOURCES="true"
      ;;
    --force-create-kind-cluster)
      FORCE_CREATE_KIND_CLUSTER="true"
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

function validate_args() {
  if [[ "${TLS_ENABLED}" == "true" ]]; then
    if [[ -z "${ETCD_CLIENT_SVC_NAME}" ]]; then
      echo -e "ETCD client service name has not been set. This is required if TLS has been enabled"
      exit 1
    fi
    if [[ -z "${ETCD_PEER_SVC_NAME}" ]]; then
      echo -e "ETCD peer service name has not been set. This is required if TLS has been enabled"
      exit 1
    fi
    if [[ -z "${TARGET_PKI_DIR}" ]]; then
      echo -e "PKI target directory has not been set. This is required if TLS has been enabled"
      exit 1
    fi
  fi
}

function check_and_create_kind_cluster() {
  local cluster_name check_cluster_exists
  cluster_name=$(get_kind_cluster_name)

  check_cluster_exists=$(kind get clusters | grep "${cluster_name}")

  if [[ "${check_cluster_exists}" != "${cluster_name}" ]]; then
    echo "cluster ${cluster_name} does not exist. Creating KIND cluster."
    create_kind_cluster "${cluster_name}"
  else
    echo "cluster ${cluster_name} already exists..."
    if [[ "${FORCE_CREATE_KIND_CLUSTER}" == "true" ]]; then
      echo "re-creating cluster ${cluster_name}..."
      delete_kind_cluster "${cluster_name}"
      create_kind_cluster "${cluster_name}"
    fi
  fi
}

function create_kind_cluster() {
  if [[ $# -ne 1 ]]; then
    echo -e "${FUNCNAME[0]} expects a cluster name"
    exit 1
  fi
  cluster_name="$1"
  # create a new KIND cluster
  echo "creating kind cluster $cluster_name"
  cat <<EOF | kind create cluster -n "${cluster_name}" --config=-
  kind: Cluster
  apiVersion: kind.x-k8s.io/v1alpha4
  featureGates:
    "StatefulSetAutoDeletePVC": true
EOF
}

function get_kind_cluster_name() {
  local cluster_name="kind"
  if [[ -n "${KIND_CLUSTER_NAME}" ]]; then
    cluster_name="${KIND_CLUSTER_NAME}"
  fi
  echo "${cluster_name}"
}

function delete_kind_cluster() {
  local cluster_name="$1"
  if [[ -z "${cluster_name}" ]]; then
    echo "cluster name should be passed for the kind delete cluster command"
    exit 1
  fi
  kind delete cluster -n "${cluster_name}"
  ret_code=$?
  if [[ "${ret_code}" != 0 ]]; then
    echo -e "failed to delete cluster $cluster_name}. Please check and re-run the script."
    exit 1
  fi
}

function create_namespace() {
  if [[ -n "${TARGET_NAMESPACE}" ]]; then
    echo "creating namespace ${TARGET_NAMESPACE}"
    cat <<EOF | kubectl apply -f -
  apiVersion: v1
  kind: Namespace
  metadata:
    labels:
      gardener.cloud/purpose: etcd-wrapper-test
    name: $TARGET_NAMESPACE
EOF
  else
    echo "using default namespace to setup etcd resources"
  fi
}

function create_pki_resources() {
  # check if all PKI resources are existing. Even if one is missing then we recreate all or if FORCE_CREATE_PKI_RESOURCES=true.
  if [[ "${TLS_ENABLED}" == "true" ]]; then
    initialize_pki_resources_arr
    local all_resources_exist target_manifest_dir
    all_resources_exist=$(all_pki_resources_exist)
    if [[ "${FORCE_CREATE_PKI_RESOURCES}" == "true" || "${all_resources_exist}" == "false" ]]; then
      pki::check_prerequisites
      pki::main "${TARGET_PKI_DIR}" "${CERT_EXPIRY}" "${ETCD_CLIENT_SVC_NAME}" "${ETCD_PEER_SVC_NAME}" "${TARGET_NAMESPACE}"
    else
      echo "skipping creation of TLS resources as they already exist"
    fi
  fi
}

function create_k8s_secrets() {
  k8s::check_prerequisites
  target_manifest_dir="${SCRIPT_DIR}"/manifests/common
  k8s::main "${TARGET_NAMESPACE}" "${TARGET_PKI_DIR}" "${target_manifest_dir}"
}

function all_pki_resources_exist() {
  local all_exists="true"
  for resource in "${PKI_RESOURCES[@]}"; do
    if [[ ! -f "${resource}" ]]; then
      all_exists="false"
      break
    fi
  done
  echo "${all_exists}"
}

function main() {
  # check pre-requisites required to run this script
  check_prerequisites
  # parse flags and validate global variables which got initialized with flag values.
  parse_flags "$@"
  validate_args
  # create kind cluster and k8s resources
  check_and_create_kind_cluster
  create_namespace
  # create certificates and keys required to enabled TLS for client and peer communication.
  create_pki_resources
  # create k8s secrets in the target namespace for the TLS resources created previously
  create_k8s_secrets
  # creates an etcd cluster (single or multi-node)
  #create_etcd_cluster
}

USAGE=$(create_usage)
main "$@"
