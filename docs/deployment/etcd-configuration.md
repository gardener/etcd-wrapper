# Configuring ETCD

There are several configuration parameters that needs to be configured to run ETCD. For a complete list of configuration options refer [official-documentation](https://etcd.io/docs/v3.5/op-guide/configuration/). To provide configuration to start the ETCD process one must create a `ConfigMap` which needs be mounted to the `backup-restore` container. `etcd-wrapper` will fetch the configuration via an internal HTTP(s) endpoint exposed out of the `backup-restore` container.

## Example ConfigMap

ConfigMap shows below is created with: `tls-disabled`, `single-node-etcd-cluster`

```yaml
apiVersion: v1
data:
  etcd.conf.yaml: |

    # Human-readable name for this member.
    name: etcd-0ef247
    # Path to the data directory.
    data-dir: /var/etcd/data/new.etcd
    # metrics configuration
    metrics: extensive
    # Number of committed transactions to trigger a snapshot to disk.
    snapshot-count: 75000

    # Accept etcd V2 client requests
    enable-v2: false
    # Raise alarms when backend size exceeds the given quota. 0 means use the
    # default quota.
    quota-backend-bytes: 8589934592
    # List of comma separated URLs to listen on for client traffic.
    listen-client-urls: http://0.0.0.0:2379

    # List of this member's client URLs to advertise to the public.
    # The URLs needed to be a comma-separated list.
    advertise-client-urls: http@etcd-main-peer@default@2379
    # List of comma separated URLs to listen on for peer traffic.
    listen-peer-urls: http://0.0.0.0:2380

    # List of this member's peer URLs to advertise to the public.
    # The URLs needed to be a comma-separated list.
    initial-advertise-peer-urls: http@etcd-main-peer@default@2380

    # Initial cluster token for the etcd cluster during bootstrap.
    initial-cluster-token: etcd-cluster

    # Initial cluster state ('new' or 'existing').
    initial-cluster-state: new

    # Initial cluster
    initial-cluster: etcd-main-0=http://etcd-main-0.etcd-main-peer.default.svc:2380

    # auto-compaction-mode ("periodic" or "revision").
    auto-compaction-mode: periodic
    # auto-compaction-retention defines Auto compaction retention length for etcd.
    auto-compaction-retention: 30m
kind: ConfigMap
metadata:
  name: etcd-main-bootstrap
  labels:
    name: etcd
    app: etcd-statefulset
    gardener.cloud/role: control-plane
    role: main
    instance: etcd-main
```

## Generation

We provide [convenience scripts](../../hack/local-dev/generate_k8s_resources.sh) which will help generate all k8s-resources required to setup etcd-wrapper. ConfigMap will get generated as part of running this script. This script is one of the many scripts used to setup a local dev-etcd-cluster on a [KIND](https://kind.sigs.k8s.io/) cluster.

> **NOTE:** To generate all resources to setup an etcd-cluster it is highly recommended that you use [druid](https://github.com/gardener/etcd-druid). Only if you wish to test `etcd-wrapper` in isolation should you depend upon the scripts in the `/hack/local-dev` folder.
