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

KIND_CLUSTER_NAME=""
FORCE_CREATE_KIND_CLUSTER="false"
FEATURE_GATES_STR=""
DELETE_CLUSTER="false"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
CONFIG_DIR="${SCRIPT_DIR}"/config

function kind::create_usage() {
  usage=$(printf '%s\n' "
  usage: $(basename $0) [Options]
  Options:
   -n | --cluster-name  <kind-cluster-name>         (Optional) name of the kind cluster. if not specified, uses default name 'kind'
   -f | --force-create                              If this flag is specified then it will always create a fresh cluster.
   -g | --feature-gates <feature gates to enable>   Comma separated list of feature gates that needs to be enabled.
   -d | --delete                                    Deletes a kind cluster. If a name is provided via '-n | --cluster-name' then it will delete that cluster else it deletes the default kind cluster with name 'kind'. If this option is not used then it will by default create a kind cluster.
   ")
  echo "${usage}"
}

function kind::check_prerequisites() {
  if ! command -v docker &>/dev/null; then
    echo -e "docker is not installed. Please install, refer: https://docs.docker.com/desktop/"
    exit 1
  fi
  if ! command -v kind &>/dev/null; then
    echo -e "kind is not installed. Please install, refer: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
    exit 1
  fi
  if ! command -v yq &>/dev/null; then
    echo -e "yq is not installed. Please install, refer: https://github.com/mikefarah/yq#install"
  fi
}

function kind::parse_flags() {
  while test $# -gt 0; do
    case "$1" in
    --cluster-name | -n)
      shift
      KIND_CLUSTER_NAME="$1"
      ;;
    --force-create | -f)
      FORCE_CREATE_KIND_CLUSTER="true"
      ;;
    --feature-gates | -g)
      shift
      FEATURE_GATES_STR="$1"
      ;;
    --delete | -d)
      DELETE_CLUSTER="true"
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

function kind::check_and_create_kind_cluster() {
  local cluster_name cluster_exists
  cluster_name=$(kind::get_kind_cluster_name)
  cluster_exists=$(kind::cluster_exists "${cluster_name}")
  if [[ "${cluster_exists}" == "false" ]]; then
    echo "cluster ${cluster_name} does not exist. Creating KIND cluster."
    kind::create_kind_cluster "${cluster_name}"
  else
    echo "cluster ${cluster_name} already exists..."
    if [[ "${FORCE_CREATE_KIND_CLUSTER}" == "true" ]]; then
      echo "re-creating cluster ${cluster_name}..."
      kind::delete_kind_cluster "${cluster_name}"
      kind::create_kind_cluster "${cluster_name}"
    fi
  fi
}

function kind::create_kind_cluster() {
  if [[ $# -ne 1 ]]; then
    echo -e "${FUNCNAME[0]} expects a cluster name"
    exit 1
  fi
  local cluster_name kind_config_path
  cluster_name="$1"
  kind_config_path="${CONFIG_DIR}"/kind.config.yaml
  cp "${CONFIG_DIR}"/kind.cluster.config.template.yaml "${kind_config_path}"

  # create a new KIND cluster
  if [[ -n "${FEATURE_GATES_STR}" ]]; then
    declare -a feature_gates
    feature_gates=($(echo "${FEATURE_GATES_STR}" | tr ',' ' '))
    for key in "${feature_gates[@]}"; do
      feature_key="$key" yq -i 'with(.featureGates.[strenv(feature_key)]; . = true | key style = "double")' "${kind_config_path}"
    done
  fi

  echo "creating kind cluster $cluster_name using config $kind_config_path"
  kind create cluster -n "${cluster_name}" --config="${kind_config_path}"
}

function kind::get_kind_cluster_name() {
  local cluster_name="kind"
  if [[ -n "${KIND_CLUSTER_NAME}" ]]; then
    cluster_name="${KIND_CLUSTER_NAME}"
  fi
  echo "${cluster_name}"
}

function kind::cluster_exists() {
  if [[ $# -ne 1 ]]; then
    echo -e "${FUNCNAME[0]} expects name of a kind cluster"
    exit 1
  fi
  local existing_clusters cluster_name exists
  cluster_name="$1"
  exists="false"
  existing_clusters=($(echo $( (kind get clusters) 2>&1) | tr '\n' ' '))
  for cluster in "${existing_clusters[@]}"; do
    if [[ "${cluster}" =~ ^"${cluster_name}"$ ]]; then
      exists="true"
      break
    fi
  done
  echo "${exists}"
}

function kind::delete_kind_cluster() {
  if [[ $# -ne 1 ]]; then
    echo -e "${FUNCNAME[0]} expects name of a kind cluster"
    exit 1
  fi
  local cluster_name="$1"
  kind delete cluster -n "${cluster_name}"
  ret_code=$?
  if [[ "${ret_code}" != 0 ]]; then
    echo -e "failed to delete cluster $cluster_name. Please check and re-run the script."
    exit 1
  fi
}

function kind::main() {
  kind::check_prerequisites
  kind::parse_flags "$@"
  if [[ "${DELETE_CLUSTER}" == "true" ]]; then
    local cluster_name cluster_exists
    cluster_name=$(kind::get_kind_cluster_name)
    cluster_exists=$(kind::cluster_exists "${cluster_name}")
    if [[ "${cluster_exists}" == "false" ]]; then
      echo -e "cluster $cluster_name does not exist. Skipping delete."
    else
      kind::delete_kind_cluster "${cluster_name}"
    fi
    return
  fi
  kind::check_and_create_kind_cluster
}

USAGE=$(kind::create_usage)
kind::main "$@"
