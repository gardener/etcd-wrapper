#!/usr/bin/env bash

function get_wrapper_process_id() {
  process_id=$(pgrep wrapper)
  echo "${process_id}"
}

wrapper_pid=$(get_wrapper_process_id)

cat <<EOF
 📌 ETCD PKI resource paths:
  --------------------------------------------------
  --cacert=proc/${wrapper_pid}/root/var/etcd/ssl/client/ca/bundle.crt
  --cert=proc/${wrapper_pid}/root/var/etcd/ssl/client/client/tls.crt
  --key=proc/${wrapper_pid}/root/var/etcd/ssl/client/client/tls.key
EOF

