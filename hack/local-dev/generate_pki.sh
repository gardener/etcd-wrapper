#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

function pki::check_prerequisites() {
  if ! command -v cfssl &>/dev/null; then
    echo -e "cfssl is not installed. Please refer: https://github.com/cloudflare/cfssl#installation"
    exit 1
  fi
  if ! command -v cfssljson &>/dev/null; then
    echo -e "cfssljson is not installed. Please refer: https://github.com/cloudflare/cfssl#installation"
    exit 1
  fi
}

##########################  CA Certificate and Key Generation functions ############################

function pki::create_ca_config() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} function expects complete target path to CA config JSON and certificate expiry"
    exit 1
  fi
  local path="$1"
  local cert_expiry="$2"
  echo "writing ${path}"
  cat >"${path}" <<EOF
  {
    "signing": {
      "default": {
        "expiry": "${cert_expiry}"
      },
      "profiles": {
        "etcd-server": {
          "expiry": "${cert_expiry}",
          "usages": [
            "signing",
            "key encipherment",
            "server auth"
          ]
        },
        "etcd-client": {
          "expiry": "${cert_expiry}",
          "usages": [
            "signing",
            "key encipherment",
            "client auth"
          ]
        },
        "etcd-peer": {
          "expiry": "${cert_expiry}",
          "usages": [
            "signing",
            "key encipherment",
            "server auth",
            "client auth"
          ]
        }
      }
    }
  }
EOF
}

function pki::generate_etcd_ca_certificate_key_pair() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} function expects PKI target directory and certificate expiry"
    exit 1
  fi
  local pki_dir="$1"
  local cert_expiry="$2"
  # create etcd CA config and CSR
  local etcd_ca_json_path="${pki_dir}"/requests/ca-config.json
  local etcd_ca_csr_path="${pki_dir}"/requests/etcd-ca-csr.json
  pki::create_ca_config "${etcd_ca_json_path}" "${cert_expiry}"
  pki::create_ca_csr_config "${etcd_ca_csr_path}" etcd-ca
  # generate etcd CA certificate and private key
  echo "> Generating ETCD CA certificate and private key at location ${pki_dir} ..."
  cfssl gencert -initca "${etcd_ca_csr_path}" | cfssljson -bare "${pki_dir}"/ca -
}

##################################################################################################

#######################  Etcd Server Certificate and Key Generation functions #####################

function pki::generate_etcd_server_certificate_key_pair() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} function expects namespace, PKI target directory and etcd client service name"
    exit 1
  fi

  local pki_dir="$1"
  local namespace="$2"
  local etcd_client_svc_name="$3"

  sans=$(pki::create_subject_alternate_names "${namespace}" "${etcd_client_svc_name}")
  pki::create_etcd_csr_config "${pki_dir}"/requests/etcd-server-csr.json "${sans}"
  echo "> Generating ETCD server certificate and private key..."
  cfssl gencert \
    -ca "${pki_dir}"/ca.pem \
    -ca-key "${pki_dir}"/ca-key.pem \
    -config "${pki_dir}"/requests/ca-config.json \
    -profile=etcd-server \
    "${pki_dir}"/requests/etcd-server-csr.json | cfssljson -bare "${pki_dir}"/etcd-server
}

##################################################################################################

#######################  Etcd Peer Server Certificate and Key Generation functions #####################

function pki::generate_etcd_peer_ca_certificate_key_pair() {
  if [[ $# -ne 1 ]]; then
    echo -e "${FUNCNAME[0]} function expects PKI target directory"
    exit 1
  fi
  local pki_dir="$1"
  local etcd_peer_ca_csr_path="${pki_dir}"/requests/etcd-peer-ca-csr.json
  pki::create_ca_csr_config "${etcd_peer_ca_csr_path}" etcd-peer-ca
  # generate etcd CA certificate and private key
  echo "> Generating ETCD Peer CA certificate and private key at location ${pki_dir} ..."
  cfssl gencert -initca "${etcd_peer_ca_csr_path}" | cfssljson -bare "${pki_dir}"/peer-ca -
}

function pki::generate_etcd_peer_certificate_key_pair() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} function expects namespace, PKI target directory and etcd peer service name"
    exit 1
  fi

  local pki_dir="$1"
  local namespace="$2"
  local etcd_peer_svc_name="$3"

  local sans_arr
  sans_arr=$(pki::create_subject_alternate_names "${namespace}" "${etcd_peer_svc_name}")
  pki::create_etcd_csr_config "${pki_dir}"/requests/etcd-peer-csr.json "${sans_arr[@]}"
  echo "> Generating ETCD peer certificate and private key..."
  cfssl gencert \
    -ca "${pki_dir}"/peer-ca.pem \
    -ca-key "${pki_dir}"/peer-ca-key.pem \
    -config "${pki_dir}"/requests/ca-config.json \
    -profile=etcd-peer \
    "${pki_dir}"/requests/etcd-peer-csr.json | cfssljson -bare "${pki_dir}"/etcd-peer
}

##################################################################################################

#######################  Etcd Client Certificate and Key Generation functions #####################

function pki::generate_etcd_client_certificate_key_pair() {
  if [[ $# -ne 1 ]]; then
    echo -e "${FUNCNAME[0]} function expects PKI target directory and etcd peer service name"
    exit 1
  fi
  local pki_dir="$1"

  pki::create_etcd_csr_config "${pki_dir}"/requests/etcd-client-csr.json ""
  echo "> Generating ETCD client certificate and private key..."
  cfssl gencert \
    -ca "${pki_dir}"/ca.pem \
    -ca-key "${pki_dir}"/ca-key.pem \
    -config "${pki_dir}"/requests/ca-config.json \
    -profile=etcd-peer \
    "${pki_dir}"/requests/etcd-client-csr.json | cfssljson -bare "${pki_dir}"/etcd-client
}

##################################################################################################

function pki::create_etcd_csr_config() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} expects a target CSR file path and an array of subject alternate names."
    exit 1
  fi
  local path="$1"
  shift
  local sans="$*"
  echo "writing ${path}"
  cat >"${path}" <<EOF
  {
    "CN": "etcd",
    "hosts": [${sans}],
    "key": {
      "algo": "rsa",
      "size": 2048
    },
    "names": [
      {
        "O": "k8s",
        "OU": "etcd",
        "L": "Walldorf",
        "ST": "BW",
        "C": "DE"
      }
    ]
  }
EOF
}

function pki::create_ca_csr_config() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} function expects complete target path to CA CSR config JSON and common-name"
    exit 1
  fi
  local path="$1"
  local common_name="$2"
  echo "writing ${path}"
  cat >"${path}" <<EOF
  {
    "CN": "${common_name}",
    "key": {
      "algo": "rsa",
      "size": 2048
    },
    "names": [
      {
        "O": "k8s",
        "OU": "etcd-cluster",
        "L": "Walldorf",
        "ST": "BW",
        "C": "DE"
      }
    ]
  }
EOF
}

function pki::create_subject_alternate_names() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} expects namespace and service name."
    exit 1
  fi

  local namespace="$1"
  local service_name="$2"
  declare -a sans_arr

  # add host names which will be added as SAN to the certificate.
  sans_arr+=("localhost")
  sans_arr+=("127.0.0.1")
  sans_arr+=($(printf "%s" "${service_name}"))
  sans_arr+=($(printf "%s.%s" "${service_name}" "${namespace}"))
  sans_arr+=($(printf "%s.%s.svc" "${service_name}" "${namespace}"))
  sans_arr+=($(printf "%s.%s.svc.cluster.local" "${service_name}" "${namespace}"))
  sans_arr+=(*.$(printf "%s" "${service_name}"))
  sans_arr+=(*.$(printf "%s.%s" "${service_name}" "${namespace}"))
  sans_arr+=(*.$(printf "%s.%s.svc" "${service_name}" "${namespace}"))
  sans_arr+=(*.$(printf "%s.%s.svc.cluster.local" "${service_name}" "${namespace}"))

  joined_sans=$(printf ",\"%s\"" "${sans_arr[@]}")
  echo "${joined_sans:1}"
}

function pki::main() {
  if [[ $# -ne 5 ]]; then
    echo -e "${FUNCNAME[0]} expects PKI target dir, certificate expiry, etcd client service name, etcd peer service name and etcd namespace"
    exit 1
  fi

  local pki_dir="$1"
  local cert_expiry="$2"
  local etcd_client_svc_name="$3"
  local etcd_peer_svc_name="$4"
  local etcd_namespace="$5"

  echo "> Creating PKI (certificates and keys) resources..."
  mkdir -p "${pki_dir}"/requests
  pki::generate_etcd_ca_certificate_key_pair "${pki_dir}" "${cert_expiry}"
  pki::generate_etcd_server_certificate_key_pair "${pki_dir}" "${etcd_namespace}" "${etcd_client_svc_name}"
  pki::generate_etcd_peer_ca_certificate_key_pair "${pki_dir}"
  pki::generate_etcd_peer_certificate_key_pair "${pki_dir}" "${etcd_namespace}" "${etcd_peer_svc_name}"
  pki::generate_etcd_client_certificate_key_pair "${pki_dir}"
}

# You can use this script stand-alone. Just uncomment the following lines or call the functions in different bash script.
#pki::main "$@"
