#!/usr/bin/env bash

function k8s::check_prerequisites() {
  # check if you have gnu base64 setup properly as that will be used to base64 encode secrets
  if ! echo "test-string" | base64 -w 0 &>/dev/null; then
    echo -e "install gnu-base64 and ensure that it is first in the PATH. Refer: https://github.com/gardener/gardener/blob/master/docs/development/local_setup.md#preparing-the-setup"
    exit 1
  fi
}

function k8s::generate_etcd_ca_secret() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} expects three arguments. Target namespace, PKI directory and target directory to store generated secret YAML"
    exit 1
  fi

  local namespace="$1"
  local pki_dir="$2"
  local target_yaml_dir="$3"

  echo "> Writing ${target_yaml_dir}/etcd-ca-secret.yaml"
  k8s::generate_ca_secret "${namespace}" "ca-etcd-bundle" "$pki_dir" "ca.pem" "${target_yaml_dir}"/etcd-ca-secret.yaml
}

function k8s::generate_etcd_peer_ca_secret() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} expects three arguments. Target namespace, PKI directory and target directory to store generated secret YAML"
    exit 1
  fi

  local namespace="$1"
  local pki_dir="$2"
  local target_yaml_dir="$3"

  echo "> Writing ${target_yaml_dir}/etcd-peer-ca-secret.yaml"
  k8s::generate_ca_secret "${namespace}" "ca-etcd-peer-bundle" "$pki_dir" "peer-ca.pem" "${target_yaml_dir}"/etcd-peer-ca-secret.yaml
}

function k8s::generate_etcd_server_secret() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} expects three arguments. Target namespace, PKI directory and target directory to store generated secret YAML"
    exit 1
  fi

  local namespace="$1"
  local pki_dir="$2"
  local target_yaml_dir="$3"

  echo "> Writing ${target_yaml_dir}/etcd-server-secret.yaml"
  k8s::generate_secret "${namespace}" "etcd-server" "${pki_dir}" "etcd-server.pem" "etcd-server-key.pem" "${target_yaml_dir}"/etcd-server-secret.yaml
}

function k8s::generate_etcd_peer_secret() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} expects three arguments. Target namespace, PKI directory and target directory to store generated secret YAML"
    exit 1
  fi

  local namespace="$1"
  local pki_dir="$2"
  local target_yaml_dir="$3"

  echo "> Writing ${target_yaml_dir}/etcd-peer-server-secret.yaml"
  k8s::generate_secret "${namespace}" "etcd-peer-server" "${pki_dir}" "etcd-peer.pem" "etcd-peer-key.pem" "${target_yaml_dir}"/etcd-peer-server-secret.yaml
}

function k8s::generate_etcd_client_secret() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} expects three arguments. Target namespace, PKI directory and target directory to store generated secret YAML"
    exit 1
  fi

  local namespace="$1"
  local pki_dir="$2"
  local target_yaml_dir="$3"

  echo "> Writing ${target_yaml_dir}/etcd-client-secret.yaml"
  k8s::generate_secret "${namespace}" "etcd-client" "${pki_dir}" "etcd-client.pem" "etcd-client-key.pem" "${target_yaml_dir}"/etcd-client-secret.yaml
}

function k8s::generate_ca_secret() {
  if [[ $# -ne 5 ]]; then
    echo -e "${FUNCNAME[0]} expects five arguments. target namespace, secret name, PKI directory, certificate filename and absolute path to the target file to store generated secret YAML"
    exit 1
  fi

  local namespace="$1"
  local secret_name="$2"
  local pki_dir="$3"
  local cert_filename="$4"
  local target_yaml_file="$5"

  echo "> Encoding CA certificate ${pki_dir}/${cert_filename}..."
  encoded_ca_cert=$(k8s::base64_encode "${pki_dir}"/"${cert_filename}")

  echo "> Writing k8s secret ${target_yaml_file}..."

  cat >"${target_yaml_file}" <<EOF
  apiVersion: v1
  kind: Secret
  metadata:
    labels:
      name: ca-etcd-peer-bundle
    name: ca-etcd-peer-bundle
    namespace: "${namespace}"
  type: Opaque
  data:
    bundle.crt: "${encoded_ca_cert}"

EOF

}

function k8s::generate_secret() {
  if [[ $# -ne 6 ]]; then
    echo -e "${FUNCNAME[0]} expects six arguments. target namespace, secret name, PKI src directory, certificate filename, key filename and absolute path to target file to store generated secret YAML"
    exit 1
  fi

  local namespace="$1"
  local secret_name="$2"
  local pki_dir="$3"
  local cert_filename="$4"
  local key_filename="$5"
  local target_yaml_file="$6"

  echo "> Encoding certificate ${pki_dir}/${cert_filename}..."
  encoded_peer_server_cert=$(k8s::base64_encode "${pki_dir}"/"${cert_filename}")
  echo "> Encoding key ${pki_dir}/${key_filename}..."
  encoded_peer_server_key=$(k8s::base64_encode "${pki_dir}"/"${key_filename}")

  echo "> Writing k8s secret ${target_yaml_file}..."

  cat >"${target_yaml_file}" <<EOF
  apiVersion: v1
  kind: Secret
  metadata:
    labels:
      name: "${secret_name}"
    name: "${secret_name}"
    namespace: "${namespace}"
  type: kubernetes.io/tls
  data:
    tls.crt: "${encoded_peer_server_cert}"
    tls.key: "${encoded_peer_server_key}"
EOF
}

function k8s::generate_etcd_configmap() {
  if [[ $# -ne 4 ]]; then
    echo -e "${FUNCNAME[0]} expects ConfigMap name, etcd instance name, source path for etcd configuration and target path to store generated ConfigMap"
    exit 1
  fi
  local name="$1"
  local etcd_instance_name="$2"
  local etcd_config_path="$3"
  local target_dir="$4"

  echo "> Writing k8s configmap ${target_dir}..."
  kubectl create configmap "${name}" --from-file "${etcd_config_path}" --dry-run=true -oyaml |
    yq 'del(.metadata.creationTimestamp)' |
    yq eval "(.metadata.labels.name = \"etcd\")
    | (.metadata.labels.app = \"etcd-statefulset\")
    | (.metadata.labels.\"gardener.cloud/role\" = \"control-plane\")
    | (.metadata.labels.role = \"main\")
    | (.metadata.labels.instance = \"${etcd_instance_name}\")
  " >"${target_dir}/${name}".yaml
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
  local namespace="$1"
  local pki_dir="$2"
  local target_dir="$3"
  echo "> Creation k8s secret YAML files..."
  # create all k8s secret yaml files
  k8s::generate_etcd_ca_secret "${namespace}" "${pki_dir}" "${target_dir}"
  k8s::generate_etcd_peer_ca_secret "${namespace}" "${pki_dir}" "${target_dir}"
  k8s::generate_etcd_server_secret "${namespace}" "${pki_dir}" "${target_dir}"
  k8s::generate_etcd_peer_secret "${namespace}" "${pki_dir}" "${target_dir}"
  k8s::generate_etcd_client_secret "${namespace}" "${pki_dir}" "${target_dir}"

  # applying all the yaml files to k8s cluster targeting the provided target namespace
  echo "> Applying all secrets to namespace ${namespace}"
  kubectl apply -f "${target_dir}"
}
