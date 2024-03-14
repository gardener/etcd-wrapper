#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0


set -o errexit
set -o nounset
set -o pipefail

# TARGET_NAMESPACE is the target namespace where the resources will be setup. If not specified then it assumes 'default' namespace is the target namespace where the resources will be setup. If not specified then it assumes 'default' namespace.
TARGET_NAMESPACE="default"
# TLS_ENABLED possible values 'true' and 'false' (default). If its value is true then TLS resources will be generated and all communication will be TLS enabled.
TLS_ENABLED="false"
ETCD_CLUSTER_SIZE=1
ETCD_INSTANCE_NAME="etcd-main"
CERT_EXPIRY="12h"
FORCE_CREATE_PKI_RESOURCES="false"
ETCD_BR_IMAGE=""
ETCD_WRAPPER_IMAGE="etcd-wrapper"
ETCD_PVC_RETAIN_POLICY="Retain"
SKAFFOLD_RUN_MODE="run"
DRY_RUN="false"

declare -a PKI_RESOURCES

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
TARGET_PKI_DIR="${SCRIPT_DIR}"/secrets
PROJECT_DIR="$(cd "$(dirname "$SCRIPT_DIR")/.." &>/dev/null && pwd)"

source "${SCRIPT_DIR}"/generate_pki.sh
source "${SCRIPT_DIR}"/generate_k8s_resources.sh

function create_usage() {
  usage=$(printf '%s\n' "
  usage: $(basename $0) [Options]
  Options:
   -n | --namespace                   <namespace>                           (Optional) kubernetes namespace where etcd resources will be created. if not specified uses 'default'
   -s | --cluster-size                <size of etcd cluster>                (Optional) size of an etcd cluster. Supported values are 1 or 3. Defaults to 1
   -t | --tls-enabled                 <is-tls-enabled>                      (Optional) controls the TLS communication amongst peers and between etcd and its client.Possible values: ['true' | 'false']. Defaults to 'false'
   -i | --etcd-instance-name          <name of etcd instance>               (Option) name of the etcd instance. Defaults to 'etcd-main'
   -e | --cert-expiry                 <certificate expiry>                  (Optional) common expiry for all certificates generated. Defaults to '12h'
   -m | --etcd-br-image               <image:tag of etcd-br container>      (Required) Image (with tag) for etcdbr container
   -w | --etcd-wrapper-image          <image of etcd-wrapper container>     (Optional) Image (without tag) for etcd-wrapper container. Skaffold will add git-commit as the tag when it builds the etcd-wrapper image.
   -r | --skaffold-run-mode           <skaffold run or debug>               (Optional) Possible values: 'run' | 'debug'. Defaults to 'run'. Will only be effective if '-d | --dry-run' is not specified.
   -f | --force-create-pki-resources                                        (Optional) If specified then it will re-create all PKI resources.
   -d | --dry-run                                                           (Optional) If set it will only generate all manifests and configuration files. The user needs to explicitly run skaffold to deploy the k8s resources.
   ")
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
    echo -e "docker is not installed. Please install, refer: https://docs.docker.com/desktop/"
    exit 1
  fi
  if ! command -v kind &>/dev/null; then
    echo -e "kind command is not found. Please install, refer: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
    exit 1
  fi
  if ! command -v skaffold &>/dev/null; then
    echo -e "skaffold is not installed. Please install, refer: https://skaffold.dev/docs/install/"
    exit 1
  fi
  if ! command -v yq &>/dev/null; then
    echo -e "yq is not installed. Please install, refer: https://github.com/mikefarah/yq#install"
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
    --cluster-size | -s)
      shift
      ETCD_CLUSTER_SIZE=$1
      ;;
    --tls-enabled | -t)
      shift
      TLS_ENABLED=$(echo "$1" | awk '{print tolower($0)}')
      ;;
    --etcd-instance-name | -i)
      shift
      ETCD_INSTANCE_NAME="$1"
      ;;
    --cert-expiry | -e)
      shift
      CERT_EXPIRY="$1"
      ;;
    --force-create-pki-resources | -f)
      FORCE_CREATE_PKI_RESOURCES="true"
      ;;
    --etcd-br-image | -m)
      shift
      ETCD_BR_IMAGE="$1"
      ;;
    --etcd-wrapper-image | -w)
      shift
      ETCD_WRAPPER_IMAGE="$1"
      ;;
    --skaffold-run-mode | -r)
      shift
      SKAFFOLD_RUN_MODE="$1"
      ;;
    --dry-run | -d)
      DRY_RUN="true"
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
  if [[ -z "${ETCD_INSTANCE_NAME}" ]]; then
    echo -e "ETCD instance name has not been set."
    exit 1
  fi
  if [[ -z "${ETCD_BR_IMAGE}" ]]; then
    echo -e "etcd-br-image is not set."
    exit 1
  fi
  # split ETCD_WRAPPER_IMAGE and ensure that only an image name is passed and no tag is passed
  IFS=':'
  read -ra wrapper_image_parts <<<"${ETCD_WRAPPER_IMAGE}"
  if [[ "${#wrapper_image_parts[@]}" -gt 1 ]]; then
    echo -e "etcd-wrapper-image should only have an image name and should not have any tags. Please remove the tag and retry."
    exit 1
  fi
  unset IFS
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
    local all_resources_exist
    all_resources_exist=$(all_pki_resources_exist)
    if [[ "${FORCE_CREATE_PKI_RESOURCES}" == "true" || "${all_resources_exist}" == "false" ]]; then
      pki::check_prerequisites
      pki::main "${TARGET_NAMESPACE}" "${ETCD_INSTANCE_NAME}" "${CERT_EXPIRY}"
    else
      echo "skipping creation of TLS resources as they already exist"
    fi
  fi
}

function create_k8s_resources() {
  k8s::check_prerequisites
  k8s::main "${ETCD_INSTANCE_NAME}" "${ETCD_CLUSTER_SIZE}" "${TLS_ENABLED}" "${ETCD_WRAPPER_IMAGE}" "${ETCD_BR_IMAGE}" "${ETCD_PVC_RETAIN_POLICY}"
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

function create_etcd_config() {
  local scheme="http"
  if [[ "${TLS_ENABLED}" == "true" ]]; then
    scheme="https"
  fi
  local etcd_namespace etcd_peer_service_name etcd_initial_cluster peer_url_scheme
  peer_url_scheme="http"
  etcd_namespace="${TARGET_NAMESPACE}"
  etcd_peer_service_name="${ETCD_INSTANCE_NAME}"-peer
  etcd_initial_cluster=$(create_etcd_config_initial_cluster)

  if [[ "${TLS_ENABLED}" == "true" && "${ETCD_CLUSTER_SIZE}" -gt 1 ]]; then
    peer_url_scheme="https"
  fi

  echo "> Creating etcd configuration at ${SCRIPT_DIR}/config/etcd.config.yaml..."
  export etcd_namespace etcd_peer_service_name scheme etcd_initial_cluster peer_url_scheme
  envsubst <"${SCRIPT_DIR}"/config/etcd.config.template.yaml >"${SCRIPT_DIR}"/config/etcd.conf.yaml
  unset etcd_namespace etcd_peer_service_name scheme etcd_initial_cluster peer_url_scheme

  if [[ "${TLS_ENABLED}" == "true" ]]; then
    yq -i \
      '(.client-transport-security.cert-file = "/var/etcd/ssl/client/server/tls.crt")
       | (.client-transport-security.key-file = "/var/etcd/ssl/client/server/tls.key")
       | (.client-transport-security.client-cert-auth = true)
       | (.client-transport-security.trusted-ca-file = "/var/etcd/ssl/client/ca/bundle.crt")
       | (.client-transport-security.auto-tls = false)' \
      "${SCRIPT_DIR}"/config/etcd.conf.yaml
    if [[ "${ETCD_CLUSTER_SIZE}" -gt 1 ]]; then
      yq -i \
        '(.peer-transport-security.cert-file = "/var/etcd/ssl/peer/server/tls.crt")
         | (.peer-transport-security.key-file = "/var/etcd/ssl/peer/server/tls.key")
         | (.peer-transport-security.client-cert-auth = true)
         | (.peer-transport-security.trusted-ca-file = "/var/etcd/ssl/peer/ca/bundle.crt")
         | (.peer-transport-security.auto-tls = false)' \
        "${SCRIPT_DIR}"/config/etcd.conf.yaml
    fi
  fi
}

function create_etcd_config_initial_cluster() {
  local scheme="http"
  if [[ "${TLS_ENABLED}" == "true" && "${ETCD_CLUSTER_SIZE}" -gt 1 ]]; then
    scheme="https"
  fi

  if [[ "${ETCD_CLUSTER_SIZE}" -gt 1 ]]; then
    for ((i = 0; i < "${ETCD_CLUSTER_SIZE}"; i++)); do
      etcd_initial_cluster+="${ETCD_INSTANCE_NAME}-${i}=${scheme}://${ETCD_INSTANCE_NAME}-${i}.${etcd_peer_service_name}.${etcd_namespace}.svc:2380,"
    done
    etcd_initial_cluster="${etcd_initial_cluster%?}"
  else
    etcd_initial_cluster="${ETCD_INSTANCE_NAME}-0=${scheme}://${ETCD_INSTANCE_NAME}-0.${etcd_peer_service_name}.${etcd_namespace}.svc:2380"
  fi
  echo "${etcd_initial_cluster}"
}

function generate_skaffold_yaml() {
  local target_path="${PROJECT_DIR}"/skaffold.yaml
  echo "> Generating Skaffold ${target_path}"
  cat >"${target_path}" <<EOF
apiVersion: skaffold/v4beta4
kind: Config
metadata:
  name: etcd-cluster
build:
  local: {}
  artifacts:
    - image: "${ETCD_WRAPPER_IMAGE}"
      ko:
        fromImage: gcr.io/distroless/static-debian11:nonroot
        dependencies:
          paths:
            - cmd
            - internal
            - go.mod
            - vendor
        flags:
          - -mod=vendor
          - -v
manifests:
  rawYaml:
    - hack/local-dev/manifests/common/backuprestore-role.rolebinding.yaml
    - hack/local-dev/manifests/common/etcd.sa.yaml
    - hack/local-dev/manifests/common/etcd-client.svc.yaml
    - hack/local-dev/manifests/common/etcd-peer.svc.yaml

EOF
  if [[ "${TLS_ENABLED}" == "true" ]]; then
    yq -i '.manifests.rawYaml += "hack/local-dev/manifests/common/ca-etcd-bundle.secret.yaml"' "${target_path}"
    yq -i '.manifests.rawYaml += "hack/local-dev/manifests/common/ca-etcd-peer-bundle.secret.yaml"' "${target_path}"
    yq -i '.manifests.rawYaml += "hack/local-dev/manifests/common/etcd-main-server.secret.yaml"' "${target_path}"
    yq -i '.manifests.rawYaml += "hack/local-dev/manifests/common/etcd-main-peer-server.secret.yaml"' "${target_path}"
    yq -i '.manifests.rawYaml += "hack/local-dev/manifests/common/etcd-client.secret.yaml"' "${target_path}"
  fi
  if [[ "${ETCD_CLUSTER_SIZE}" -gt 1 ]]; then
    yq -i '.manifests.rawYaml += "hack/local-dev/manifests/multinode/lease.yaml"' "${target_path}"
    yq -i '.manifests.rawYaml += "hack/local-dev/manifests/multinode/etcd.sts.yaml"' "${target_path}"
    yq -i '.manifests.rawYaml += "hack/local-dev/manifests/multinode/etcd-main-bootstrap.cm.yaml"' "${target_path}"
    #    yq -i '.manifests.rawYaml += "hack/local-dev/manifests/multinode/etcd-peer.svc.yaml"' "${target_path}"
  else
    yq -i '.manifests.rawYaml += "hack/local-dev/manifests/singlenode/lease.yaml"' "${target_path}"
    yq -i '.manifests.rawYaml += "hack/local-dev/manifests/singlenode/etcd.sts.yaml"' "${target_path}"
    yq -i '.manifests.rawYaml += "hack/local-dev/manifests/singlenode/etcd-main-bootstrap.cm.yaml"' "${target_path}"
  fi
}

function main() {
  # check pre-requisites required to run this script
  check_prerequisites
  # parse flags and validate global variables which got initialized with flag values.
  parse_flags "$@"
  validate_args
  create_pki_resources
  create_etcd_config
  create_k8s_resources
  generate_skaffold_yaml
  if [[ "${DRY_RUN}" == "false" ]]; then
    create_namespace
    skaffold "${SKAFFOLD_RUN_MODE}" -n "${TARGET_NAMESPACE}"
    echo "> Successfully create etcd resource in namespace: ${TARGET_NAMESPACE}"
  else
    cat <<EOF

📌 NOTE:
---------------------------------------------------------------------------
All resources are generated successfully.
A 'skaffold.yaml' file is generated and put in the root of the project folder.
Ensure your current-context is targeting the k8s cluster that you wish to use.
To deploy these resources execute 'skaffold run -n <target-namespace>'
If you are actively developing then you can also use 'dev' run-mode offered by 'skaffold'.
To do that execute 'skaffold dev -n <target-namespace>"

EOF
  fi
}

USAGE=$(create_usage)
main "$@"
