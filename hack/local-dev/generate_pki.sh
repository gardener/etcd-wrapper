#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
SECRETS_DIR="${SCRIPT_DIR}"/secrets
SECRET_REQUESTS_DIR=${SECRETS_DIR}/requests

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

function pki::create_ca_config() {
  if [[ $# -ne 1 ]]; then
    echo -e "${FUNCNAME[0]} function expects a certificate expiry"
    exit 1
  fi
  local cert_expiry="$1"
  local path="${SECRET_REQUESTS_DIR}"/ca-config.json
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

function pki::create_ca_csr_config() {
  if [[ $# -ne 1 ]]; then
    echo -e "${FUNCNAME[0]} function expects common-name"
    exit 1
  fi
  local common_name="$1"
  local path="${SECRET_REQUESTS_DIR}/${common_name}"-csr.json
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


function pki::generate_etcd_ca_certificate_key_pair() {
  if [[ $# -ne 1 ]]; then
    echo -e "${FUNCNAME[0]} function expects a certificate expiry"
    exit 1
  fi
  local cert_expiry="$1"
  # create etcd CA config and CSR
  pki::create_ca_config "${cert_expiry}"
  pki::create_ca_csr_config "etcd-ca"
  # generate etcd CA certificate and private key
  echo "> Generating ETCD CA certificate and private key at location ${SECRETS_DIR} ..."
  cfssl gencert -initca "${SECRET_REQUESTS_DIR}"/etcd-ca-csr.json | cfssljson -bare "${SECRETS_DIR}"/ca -
}

function pki::generate_etcd_server_certificate_key_pair() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} function expects namespace and etcd name"
    exit 1
  fi

  local namespace="$1"
  local etcd_name="$2"

  sans_arr=$(pki::create_etcd_server_sans "${namespace}" "${etcd_name}")
  pki::create_etcd_csr_config "${SECRET_REQUESTS_DIR}"/etcd-server-csr.json "etcd-server" "${sans_arr[*]}"
  echo "> Generating ETCD server certificate and private key..."
  cfssl gencert \
    -ca "${SECRETS_DIR}"/ca.pem \
    -ca-key "${SECRETS_DIR}"/ca-key.pem \
    -config "${SECRET_REQUESTS_DIR}"/ca-config.json \
    -profile=etcd-server \
    "${SECRET_REQUESTS_DIR}"/etcd-server-csr.json | cfssljson -bare "${SECRETS_DIR}"/etcd-server
}

function pki::generate_etcd_peer_ca_certificate_key_pair() {
  pki::create_ca_csr_config "etcd-peer-ca"
  # generate etcd CA certificate and private key
  echo "> Generating ETCD Peer CA certificate and private key at location ${SECRETS_DIR} ..."
  cfssl gencert -initca "${SECRET_REQUESTS_DIR}"/etcd-peer-ca-csr.json | cfssljson -bare "${SECRETS_DIR}"/peer-ca -
}

function pki::generate_etcd_peer_certificate_key_pair() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} function expects namespace and etcd name"
    exit 1
  fi

  local namespace="$1"
  local etcd_name="$2"

  local sans_arr
  sans_arr=$(pki::create_etcd_peer_sans "${namespace}" "${etcd_name}")
  pki::create_etcd_csr_config "${SECRET_REQUESTS_DIR}"/etcd-peer-csr.json "etcd-server" "${sans_arr[@]}"
  echo "> Generating ETCD peer certificate and private key..."
  cfssl gencert \
    -ca "${SECRETS_DIR}"/peer-ca.pem \
    -ca-key "${SECRETS_DIR}"/peer-ca-key.pem \
    -config "${SECRET_REQUESTS_DIR}"/ca-config.json \
    -profile=etcd-peer \
    "${SECRET_REQUESTS_DIR}"/etcd-peer-csr.json | cfssljson -bare "${SECRETS_DIR}"/etcd-peer
}

function pki::generate_etcd_client_certificate_key_pair() {
  pki::create_etcd_csr_config "${SECRET_REQUESTS_DIR}"/etcd-client-csr.json "etcd-client" ""
  echo "> Generating ETCD client certificate and private key..."
  cfssl gencert \
    -ca "${SECRETS_DIR}"/ca.pem \
    -ca-key "${SECRETS_DIR}"/ca-key.pem \
    -config "${SECRET_REQUESTS_DIR}"/ca-config.json \
    -profile=etcd-peer \
    "${SECRET_REQUESTS_DIR}"/etcd-client-csr.json | cfssljson -bare "${SECRETS_DIR}"/etcd-client
}

##################################################################################################

function pki::create_etcd_csr_config() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} expects a target CSR file path, common name and an array of subject alternate names."
    exit 1
  fi
  local path="$1"
  local common_name="$2"
  shift
  shift
  local sans="$*"
  echo "writing ${path}"
  cat >"${path}" <<EOF
  {
    "CN": "${common_name}",
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

function pki::create_etcd_server_sans() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} expects namespace and etcd name"
    exit 1
  fi

  local namespace="$1"
  local etcd_name="$2"
  declare -a sans_arr

  # add host names which will be added as SAN to the certificate.
  sans_arr+=($(printf "%s-local" "${etcd_name}"))
  sans_arr+=($(printf "%s-client" "${etcd_name}"))
  sans_arr+=($(printf "%s-client.%s" "${etcd_name}" "${namespace}"))
  sans_arr+=($(printf "%s-client.%s.svc" "${etcd_name}" "${namespace}"))
  sans_arr+=($(printf "%s-client.%s.svc.cluster.local" "${etcd_name}" "${namespace}"))
  sans_arr+=($(printf "*.%s-peer" "${etcd_name}"))
  sans_arr+=($(printf "*.%s-peer.%s" "${etcd_name}" "${namespace}"))
  sans_arr+=($(printf "*.%s-peer.%s.svc" "${etcd_name}" "${namespace}"))
  sans_arr+=($(printf "*.%s-peer.%s.svc.cluster.local" "${etcd_name}" "${namespace}"))

  joined_sans=$(printf ",\"%s\"" "${sans_arr[@]}")
  echo "${joined_sans:1}"
}

function pki::create_etcd_peer_sans() {
  if [[ $# -ne 2 ]]; then
    echo -e "${FUNCNAME[0]} expects namespace and etcd name"
    exit 1
  fi

  local namespace="$1"
  local etcd_name="$2"
  declare -a sans_arr

  # add host names which will be added as SAN to the certificate.
  sans_arr+=($(printf "%s-peer" "${etcd_name}"))
  sans_arr+=($(printf "%s-peer.%s" "${etcd_name}" "${namespace}"))
  sans_arr+=($(printf "%s-peer.%s.svc" "${etcd_name}" "${namespace}"))
  sans_arr+=($(printf "%s-peer.%s.svc.cluster.local" "${etcd_name}" "${namespace}"))
  sans_arr+=($(printf "*.%s-peer" "${etcd_name}"))
  sans_arr+=($(printf "*.%s-peer.%s" "${etcd_name}" "${namespace}"))
  sans_arr+=($(printf "*.%s-peer.%s.svc" "${etcd_name}" "${namespace}"))
  sans_arr+=($(printf "*.%s-peer.%s.svc.cluster.local" "${etcd_name}" "${namespace}"))

  joined_sans=$(printf ",\"%s\"" "${sans_arr[@]}")
  echo "${joined_sans:1}"
}

function pki::main() {
  if [[ $# -ne 3 ]]; then
    echo -e "${FUNCNAME[0]} expects namespace, etcd name and certificate expiry"
    exit 1
  fi

  local namespace="$1"
  local etcd_name="$2"
  local cert_expiry="$3"

  echo "> Creating PKI (certificates and keys) resources..."
  mkdir -p "${SECRET_REQUESTS_DIR}"
  pki::generate_etcd_ca_certificate_key_pair "${cert_expiry}"
  pki::generate_etcd_peer_ca_certificate_key_pair
  pki::generate_etcd_server_certificate_key_pair "${namespace}" "${etcd_name}"
  pki::generate_etcd_peer_certificate_key_pair "${namespace}" "${etcd_name}"
  pki::generate_etcd_client_certificate_key_pair
}

# You can use this script stand-alone. Just uncomment the following lines or call the functions in different bash script.
#pki::main "$@"
