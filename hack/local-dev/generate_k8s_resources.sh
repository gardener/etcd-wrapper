#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"

# It holds the value of the target directory for manifests for a single/multi-node etcd cluster.
MANIFEST_DIR="${SCRIPT_DIR}"/manifests
TEMPLATE_DIR="${SCRIPT_DIR}"/manifests/templates
PKI_DIR="${SCRIPT_DIR}"/secrets
CONFIG_DIR="${SCRIPT_DIR}"/config
ETCD_CONFIGMAP_SUFFIX="-bootstrap"

declare -a TEMPLATE_PREREQ_RESOURCE_ARR

function k8s::initialize_resource_array() {
  TEMPLATE_PREREQ_RESOURCE_ARR+=("etcd.sa.template.yaml,etcd.sa.yaml,common")
  TEMPLATE_PREREQ_RESOURCE_ARR+=("backuprestore.role.rolebinding.template.yaml,backuprestore-role.rolebinding.yaml,common")
  TEMPLATE_PREREQ_RESOURCE_ARR+=("etcd.client.svc.template.yaml,etcd-client.svc.yaml,common")
  TEMPLATE_PREREQ_RESOURCE_ARR+=("etcd.peer.svc.template.yaml,etcd-peer.svc.yaml,common")
}

function k8s::check_prerequisites() {
  # check if you have gnu base64 setup properly as that will be used to base64 encode secrets
  if ! echo "test-string" | base64 -w 0 &>/dev/null; then
    echo -e "install gnu-base64 and ensure that it is first in the PATH. Refer: https://github.com/gardener/gardener/blob/master/docs/development/local_setup.md#preparing-the-setup"
    exit 1
  fi
  if ! command -v yq &>/dev/null; then
    echo -e "yq is not installed. Please install, refer: https://github.com/mikefarah/yq#install"
    exit 1
  fi
  if ! command -v kubectl &>/dev/null; then
    echo -e "kubectl is not installed. Please install, refer: https://kubernetes.io/docs/tasks/tools/"
    exit 1
  fi
}

function k8s::generate_ca_secret() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} expects two arguments - secret name, certificate file name"
    exit 1
  fi

  local secret_name="$1"
  local cert_filename="$2"

  source_cert_path="${PKI_DIR}"/"${cert_filename}"
  target_secret_path="${MANIFEST_DIR}"/common/"${secret_name}".secret.yaml

  echo "> Encoding CA certificate ${source_cert_path}..."
  encoded_ca_cert=$(k8s::base64_encode "${source_cert_path}")

  echo "> Writing k8s secret ${target_secret_path}..."

  cat >"${target_secret_path}" <<EOF
apiVersion: v1
kind: Secret
metadata:
  labels:
    name: "${secret_name}"
  name: "${secret_name}"
type: Opaque
data:
  bundle.crt: "${encoded_ca_cert}"

EOF
}

function k8s::generate_secret() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} expects three arguments - secret name, certificate filename and key filename"
    exit 1
  fi

  local secret_name cert_filename key_filename target_secret_path encoded_cert encoded_key source_cert_path source_key_path

  secret_name="$1"
  cert_filename="$2"
  key_filename="$3"
  source_cert_path="${PKI_DIR}"/"${cert_filename}"
  source_key_path="${PKI_DIR}"/"${key_filename}"
  target_secret_path="${MANIFEST_DIR}"/common/"${secret_name}".secret.yaml

  echo "> Encoding certificate ${source_cert_path}..."
  encoded_cert=$(k8s::base64_encode "${source_cert_path}")
  echo "> Encoding key ${source_key_path}..."
  encoded_key=$(k8s::base64_encode "${source_key_path}")

  echo "> Writing k8s manifest ${target_secret_path}..."

  cat >"${target_secret_path}" <<EOF
apiVersion: v1
kind: Secret
metadata:
  labels:
    name: "${secret_name}"
  name: "${secret_name}"
type: kubernetes.io/tls
data:
  tls.crt: "${encoded_cert}"
  tls.key: "${encoded_key}"
EOF
}

function k8s::generate_lease() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} expects etcd name etcd cluster size and etcd cluster type"
    exit 1
  fi

  local etcd_name etcd_cluster_size target_path
  etcd_name="$1"
  etcd_cluster_size="$2"
  etcd_cluster_type="$3"

  target_path="${MANIFEST_DIR}/${etcd_cluster_type}"/lease.yaml
  echo "> Generating k8s manifest $target_path..."
  for ((i = 0; i < "${etcd_cluster_size}"; i++)); do
    cat <<EOF >>"${target_path}"
apiVersion: coordination.k8s.io/v1
kind: Lease
metadata:
  labels:
    instance: "${etcd_name}"
    name: etcd
  name: "${etcd_name}-${i}"

---

EOF
  done
}

# generates K8S YAML files for ServiceAccount, ETCD Client and Peer service, Role and Rolebindings and leases.
function k8s::substitute_name_and_generate_resources() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} expects etcd name and etcd cluster type (singlenode | multinode)"
    exit 1
  fi
  local etcd_name template_path target_path entry_parts resource_category
  etcd_name="$1"
  etcd_cluster_type="$2"
  export etcd_name
  for entry in "${TEMPLATE_PREREQ_RESOURCE_ARR[@]}"; do
    entry_parts=($(echo $entry | tr ',' ' '))
    resource_category="${entry_parts[2]}"
    if [[ "${resource_category}" == "common" || "${resource_category}" == "${etcd_cluster_type}" ]]; then
      template_path="${TEMPLATE_DIR}/${entry_parts[0]}"
      target_dir="${MANIFEST_DIR}/${resource_category}"
      mkdir -p "${target_dir}"
      target_path="${target_dir}/${entry_parts[1]}"
      echo "> Generating k8s manifest $target_path..."
      envsubst <"${template_path}" >"${target_path}"
    fi
  done
  unset etcd_name
}

function k8s::generate_statefulset() {
  if [[ $# -ne 7 ]]; then
    echo -e "${FUNCNAME[0]} expects seven arguments - etcd name, etcd cluster size, etcd cluster type, tls-enabled, etcd-wrapper image, etcd-br image, etcd-pvc retain policy"
    exit 1
  fi

  local etcd_name="$1"
  local etcd_cluster_size=$2
  local etcd_cluster_type="$3"
  local tls_enabled="$4"
  local etcd_wrapper_image="$5"
  local etcd_br_image="$6"
  local etcd_pvc_retain_policy="$7"

  local template_path target_path etcd_cm_name scheme
  etcd_cm_name="${etcd_name}${ETCD_CONFIGMAP_SUFFIX}"
  template_path="${MANIFEST_DIR}"/templates/etcd.sts.template.yaml
  target_path="${MANIFEST_DIR}/${etcd_cluster_type}"/etcd.sts.yaml
  scheme="http"
  if [[ "${tls_enabled}" == "true" ]]; then
    scheme="https"
  fi
  echo "> Generating k8s manifest ${target_path}..."
  export etcd_name etcd_cluster_size etcd_wrapper_image etcd_br_image tls_enabled etcd_pvc_retain_policy etcd_cm_name scheme
  envsubst <"${template_path}" >"${target_path}"

  update_sts_volume_mounts "${tls_enabled}" "${etcd_cluster_size}" "${target_path}"
  update_sts_volumes "${etcd_name}" "${tls_enabled}" "${etcd_cluster_size}" "${target_path}"
  update_sts_br_command "${tls_enabled}" "${target_path}"
  update_sts_etcd_wrapper_args "${etcd_name}" "${tls_enabled}" "${target_path}"

  unset etcd_name etcd_cluster_size etcd_wrapper_image etcd_br_image tls_enabled etcd_pvc_retain_policy etcd_cm_name scheme
}

function update_sts_br_command() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} expects two arguments: tls enabled bool and target path"
    exit 1
  fi
  local tls_enabled="$1"
  local target_path="$2"

  if [[ "${tls_enabled}" == "true" ]]; then
    yq -i '.spec.template.spec.containers[1].command += "--server-cert=/var/etcd/ssl/client/server/tls.crt"' "${target_path}"
    yq -i '.spec.template.spec.containers[1].command += "--server-key=/var/etcd/ssl/client/server/tls.key"' "${target_path}"
    yq -i '.spec.template.spec.containers[1].command += "--cert=/var/etcd/ssl/client/client/tls.crt"' "${target_path}"
    yq -i '.spec.template.spec.containers[1].command += "--key=/var/etcd/ssl/client/client/tls.key"' "${target_path}"
    yq -i '.spec.template.spec.containers[1].command += "--insecure-skip-tls-verify=false"' "${target_path}"
    yq -i '.spec.template.spec.containers[1].command += "--insecure-transport=false"' "${target_path}"
    yq -i '.spec.template.spec.containers[1].command += "--cacert=/var/etcd/ssl/client/ca/bundle.crt"' "${target_path}"
  else
    yq -i '.spec.template.spec.containers[1].command += "--insecure-skip-tls-verify=true"' "${target_path}"
    yq -i '.spec.template.spec.containers[1].command += "--insecure-transport=true"' "${target_path}"
  fi
}

function update_sts_volumes() {
  if [[ $# -ne 4 ]]; then
    echo -e "${FUNCNAME[0]} expects four arguments: etcd name, tls enabled bool, cluster size and target path"
    exit 1
  fi
  local etcd_name="$1"
  local tls_enabled="$2"
  local etcd_cluster_size="$3"
  local target_path="$4"

  if [[ "${tls_enabled}" == "true" ]]; then
    if [[ "${etcd_cluster_size}" -gt 1 ]]; then
      # add volumes for peer secrets
      yq -i '.spec.template.spec.volumes += {"name": "etcd-peer-ca", "secret": {"defaultMode": 420, "secretName": "ca-etcd-peer-bundle"}}' "${target_path}"
      etcd_peer_server_secret_name="${etcd_name}-peer-server" yq -i '.spec.template.spec.volumes += {"name": "etcd-peer-server", "secret": {"defaultMode": 420, "secretName": env(etcd_peer_server_secret_name)}}' "${target_path}"
    fi
    # add volumes for secrets
    yq -i '.spec.template.spec.volumes += {"name": "etcd-client-ca", "secret": {"defaultMode": 420, "secretName": "ca-etcd-bundle"}}' "${target_path}"
    etcd_client_server_secret_name="${etcd_name}-server" yq -i '.spec.template.spec.volumes += {"name": "etcd-client-server", "secret": {"defaultMode": 420, "secretName": env(etcd_client_server_secret_name)}}' "${target_path}"
    yq -i '.spec.template.spec.volumes += {"name": "etcd-client-client", "secret": {"defaultMode": 420, "secretName": "etcd-client"}}' "${target_path}"
  fi
}

function update_sts_volume_mounts() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} expects three arguments: tls enabled bool, cluster size and target path"
    exit 1
  fi
  local tls_enabled="$1"
  local etcd_cluster_size="$2"
  local target_path="$3"

  if [[ "${tls_enabled}" == "true" ]]; then
    # add TLS volume mounts to etcd-container
    yq -i '.spec.template.spec.containers[0].volumeMounts += {"mountPath": "/var/etcd/ssl/client/ca", "name": "etcd-client-ca"}' "${target_path}"
    yq -i '.spec.template.spec.containers[0].volumeMounts += {"mountPath": "/var/etcd/ssl/client/server", "name": "etcd-client-server"}' "${target_path}"
    yq -i '.spec.template.spec.containers[0].volumeMounts += {"mountPath": "/var/etcd/ssl/client/client", "name": "etcd-client-client"}' "${target_path}"
    # add TLS volume mounts to backup-restore
    yq -i '.spec.template.spec.containers[1].volumeMounts += {"mountPath": "/var/etcd/ssl/client/ca", "name": "etcd-client-ca"}' "${target_path}"
    yq -i '.spec.template.spec.containers[1].volumeMounts += {"mountPath": "/var/etcd/ssl/client/server", "name": "etcd-client-server"}' "${target_path}"
    yq -i '.spec.template.spec.containers[1].volumeMounts += {"mountPath": "/var/etcd/ssl/client/client", "name": "etcd-client-client"}' "${target_path}"
    # add TLS volume mounts to secure peer communication
    if [[ "${etcd_cluster_size}" -gt 1 ]]; then
      yq -i '.spec.template.spec.containers[0].volumeMounts += {"mountPath": "/var/etcd/ssl/peer/ca", "name": "etcd-peer-ca"}' "${target_path}"
      yq -i '.spec.template.spec.containers[0].volumeMounts += {"mountPath": "/var/etcd/ssl/peer/server", "name": "etcd-peer-server"}' "${target_path}"
      yq -i '.spec.template.spec.containers[1].volumeMounts += {"mountPath": "/var/etcd/ssl/peer/ca", "name": "etcd-peer-ca"}' "${target_path}"
      yq -i '.spec.template.spec.containers[1].volumeMounts += {"mountPath": "/var/etcd/ssl/peer/server", "name": "etcd-peer-server"}' "${target_path}"
    fi
  fi
}

function update_sts_etcd_wrapper_args() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} expects three arguments: etc name, tls enabled bool and target YAML path"
    exit 1
  fi
  local etcd_name tls_enabled args path_env
  etcd_name="$1"
  tls_enabled="$2"
  target_path="$3"
  path_env=".spec.template.spec.containers[0].args"

  if [[ "${tls_enabled}" == "true" ]]; then
    args="[start-etcd, '--backup-restore-tls-enabled=true', '--backup-restore-host-port=${etcd_name}-local:8080', '--etcd-server-name=${etcd_name}-local', '--etcd-client-cert-path=/var/etcd/ssl/client/client/tls.crt', '--etcd-client-key-path=/var/etcd/ssl/client/client/tls.key', '--backup-restore-ca-cert-bundle-path=/var/etcd/ssl/client/ca/bundle.crt']"
  else
    args="[start-etcd, '--backup-restore-tls-enabled=false', '--backup-restore-host-port=${etcd_name}-local:8080']"
  fi
  export path_env args
  yq -i 'eval(strenv(path_env)) = env(args)' "${target_path}"
  unset path_env args
}

function k8s::generate_etcd_configmap() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} expects etcd name and target dir to store generated ConfigMap"
    exit 1
  fi
  local etcd_name="$1"
  local target_dir="$2"
  local etcd_configmap_name="${etcd_name}${ETCD_CONFIGMAP_SUFFIX}"
  local etcd_config_path="${CONFIG_DIR}"/etcd.conf.yaml
  local etcd_configmap_path="${target_dir}/${etcd_configmap_name}".cm.yaml

  echo "> Generating k8s manifest ${etcd_configmap_path}..."
  kubectl create configmap "${etcd_configmap_name}" --from-file "${etcd_config_path}" --dry-run=client -oyaml |
    yq 'del(.metadata.creationTimestamp)' |
    yq eval "(.metadata.labels.name = \"etcd\")
      | (.metadata.labels.app = \"etcd-statefulset\")
      | (.metadata.labels.\"gardener.cloud/role\" = \"control-plane\")
      | (.metadata.labels.role = \"main\")
      | (.metadata.labels.instance = \"${etcd_name}\")" >"${etcd_configmap_path}"
}

function get_etcd_cluster_type() {
  if [[ $# -ne 1 ]]; then
    echo -e "${FUNCNAME[0]} expects etcd cluster size"
    exit 1
  fi
  local etcd_cluster_size="$1"
  local etcd_cluster_type="singlenode"
  if [[ "${etcd_cluster_size}" -gt 1 ]]; then
    etcd_cluster_type="multinode"
  fi
  echo "${etcd_cluster_type}"
}

function k8s::base64_encode() {
  if [[ $# -ne 1 ]]; then
    echo -e "k8s::base64_encode expects one argument which is the target file path to be base64 encoded"
    exit 1
  fi
  local target_file="$1"
  encoded=$(base64 -w 0 <"${target_file}")
  echo "$encoded"
}

function k8s::main() {

  if [[ $# -ne 6 ]]; then
    echo -e "${FUNCNAME[0]} expects 5 arguments - etcd name, etcd cluster size, tls enabled, etcd wrapper image, etcd-br image, etcd pvc retain policy"
    exit 1
  fi

  local etcd_name="$1"
  local etcd_cluster_size=$2
  local tls_enabled="$3"
  local etcd_wrapper_image="$4"
  local etcd_br_image="$5"
  local etcd_pvc_retain_policy="$6"

  mkdir -p "${MANIFEST_DIR}"/common
  if [[ "${tls_enabled}" == "true" ]]; then
    echo "> Creation k8s secret YAML files..."
    # create all k8s secret yaml files
    k8s::generate_ca_secret "ca-etcd-bundle" "ca.pem"
    k8s::generate_ca_secret "ca-etcd-peer-bundle" "peer-ca.pem"
    k8s::generate_secret "${etcd_name}"-server "etcd-server.pem" "etcd-server-key.pem"
    k8s::generate_secret "${etcd_name}"-peer-server "etcd-peer.pem" "etcd-peer-key.pem"
    k8s::generate_secret "etcd-client" "etcd-client.pem" "etcd-client-key.pem"
  fi

  k8s::initialize_resource_array
  local etcd_cluster_type
  etcd_cluster_type=$(get_etcd_cluster_type "${etcd_cluster_size}")
  mkdir -p "${MANIFEST_DIR}"/"${etcd_cluster_type}"

  k8s::substitute_name_and_generate_resources "${etcd_name}" "${etcd_cluster_type}"
  k8s::generate_lease "${etcd_name}" "${etcd_cluster_size}" "${etcd_cluster_type}"
  k8s::generate_etcd_configmap "${etcd_name}" "${MANIFEST_DIR}/${etcd_cluster_type}"
  k8s::generate_statefulset "${etcd_name}" "${etcd_cluster_size}" "${etcd_cluster_type}" "${tls_enabled}" "${etcd_wrapper_image}" "${etcd_br_image}" "${etcd_pvc_retain_policy}"
}

#k8s::main "$@"
