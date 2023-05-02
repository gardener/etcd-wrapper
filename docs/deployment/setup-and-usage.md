# Deploying etcd-wrapper

## Command line arguments

| Flag Name                   | Type          | Required | Default Value | Description                                                                                                                                                                                 |
| --------------------------- | ------------- | -------- | ------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| tls-enabled                 | bool          | No       | false         | Enables TLS for the application                                                                                                                                                             |
| sidecar-host-port           | string        | No       | :8080         | Host address and port of the backup restore sidecar with which this container will interact during initialization. Should be of the format <host>:<port> and must not include the protocol. |
| sidecar-ca-cert-bundle-path | string        | No       | ""            | Path of CA cert bundle (This will be used when TLS is enabled via tls-enabled flag.                                                                                                         |
| etcd-wait-ready-timeout     | time.duration | No       | 0s            | time duration the application will wait for etcd to get ready, by default it waits forever.                                                                                                 |

## Deploying etcd-wrapper in a local kind cluster

etcd-wrapper can be run as a container on a cluster. It has a dependency and is designed to run in tandem with [etcd-backup-restore](https://github.com/gardener/etcd-backup-restore) as a sidecar container within a single pod.

To try out etcd-wrapper along with etcd-backup-restore, we have provided k8s resource yaml files that you can apply to a running kubernetes cluster.

In this section, we show how to run etcd-wrapper on a local [kind](https://kind.sigs.k8s.io/) cluster. These steps can however be replicated as is on any other kubernetes cluster without modification
<br /> Please follow the [installation guide](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) to get kind installed

> We assume you already have a development environment set up. If not, please follow steps on [setting up your development environment](../development/contribution.md#setting-up-development-environment)

> *Note: we do not have a registry to host etcd-wrapper images yet (don't worry, we will soon, and we will update this doc as soon as that happens), so you will have to build an image and use your own personal docker registry to host that image*

### Create a kind cluster

1. Create cluster
   
   ```bash
     kind create cluster
   ```

2. Set kubectl context
   
   ```bash
     kubectl cluster-info --context kind-kind
   ```

You now have a kind cluster ready and can proceed to building an image and deploying that onto the cluster

### Tag, build and push the image

1. Tag the image you're about to build using the following
   
   ```
     export IMG=<registry_name>/<repo_name>:<version>
   ```

2. Build the image
   
   ```bash
     docker build . -t ${IMG}
   ```

3. Push the image
   
   > Optional: If you are using any dedicated [kind – Local Registry](https://kind.sigs.k8s.io/docs/user/local-registry/) or any remote docker registry then you should push the image to that repository and also change the StatefulSet to use these image(s).
   
   ```
     docker push <registry_name>/<repo_name>:<version>
   ```

### Deploy application

There are two variants in which the etcd cluster can be setup:

* `single-node` - in this variant a single member etcd cluster is setup.

* `multi-node` - in this variant by default a 3 member etcd cluster is setup.

There are two ways to deploy all the K8S resource required for `etcd-wrapper` to run

* [Skaffold](https://skaffold.dev/) can be used to easily setup all required resources.
  
  > It is assumed that you have already installed skaffold. If not already done so then please follow the [installation instructions](https://skaffold.dev/docs/install/).

* Manually setup each resource using `kubectl`.



#### Deploy a single-node etcd cluster:

_Setup using skaffold_

```bash
skaffold run --module single-node-etcd
```

_Manually create K8S resources_

1. Create role and rolebinding required by etcd-backup-restore container.
   
   ```bash
   kubectl apply -f example/common/backuprestore-role-rolebinding.yaml
   ```

2. Create client service. 
   
   > This will be required by clients like kube-api-server to connect to the etcd service.
   
   ```bash
   kubectl apply -f example/common/etcd-client-svc.yaml
   ```

3. Create a ServiceAccount
   
   ```bash
   kubectl apply -f example/common/etcd-sa.yaml
   ```

4. Create Secrets
   
   ```yaml
   kubectl apply -f example/common/etcd-secrets.yaml
   ```

5. Create Leases
   
   > A lease is created for each etcd member. etcd-backup-restore periodically renews its lease. An operator can use the lease to check for active/live members.
   
   ```yaml
   kubectl apply -f example/singlenode/leases.yaml
   ```

6. Create etcd ConfigMap
   
   > All configuration required by the etcd process to start is stored in this configmap
   
   ```yaml
   kubectl apply -f example/singlenode/etcd-cm.yaml
   ```

7. Create StatefulSet
   
   ```yaml
   kubectl apply -f example/singleode/etcd-sts.yaml
   ```



#### Deploy a multi-node etcd cluster:

*Setup using skaffold*

```bash
skaffold run --module multi-node-etcd
```

__Manually create K8S resources_

- Create role and rolebinding required by etcd-backup-restore container.
  
  ```bash
  kubectl apply -f example/common/backuprestore-role-rolebinding.yaml
  ```

- Create client service.
  
  > This will be required by clients like kube-api-server to connect to the etcd service.
  
  ```bash
  kubectl apply -f example/common/etcd-client-svc.yaml
  ```

- Create a ServiceAccount
  
  ```bash
  kubectl apply -f example/common/etcd-sa.yaml
  ```

- Create Secrets
  
  ```bash
  kubectl apply -f example/common/etcd-secrets.yaml
  ```

- Create Leases
  
  ```yaml
  kubectl apply -f example/multinode/leases.yaml
  ```

- Create ConfigMap
  
  ```bash
  kubectl apply -f example/multinode/etcd-configmap.yaml
  ```

- Create Peer Service
  
  > Peer service is a headless service which will used for inter pod communication between all etcd members in a StatefulSet.
  
  ```bash
  kubectl apply -f example/multinode/etcd-peer-svc.yaml
  ```

- Create Peer NetworkPolicy
  
  > Allows communication between peer pods
  
  ```bash
  kubectl apply -f example/multinode/etcd-peer-netpol.yaml
  ```

- Create StatefulSet
  
  ```bash
  kubectl apply -f example/multinode/etcd-sts.yaml
  ```

> If you change the single-node cluster to a multi-node then ensure that you manually delete all the `PersistentVolumeClaims` that gets created before you migrate.

### Cleanup

> 

When you are done, you can clean up the entire setup by deleting the kind cluster

```bash
kind delete cluster
```

## Setup TLS in etcd-wrapper

The steps outlined above sets up etcd *without* TLS

To use etcd-wrapper with TLS, follow the following steps

> The steps assume that the same CA is used to generate all secrets. If you prefer using different CAs, please pass the CAs accordingly

#### Setup secrets

1. Use the script in `hack/pki_gen.sh` can be used to generate test secrets
   
   The same secrets can be used for etcd as well as backup-restore. However, you can also go ahead and generate a different set of secrets for etcd-backup-restore if you desire.
   
   > If you prefer manually creating secrets for testing, please see [here](https://github.com/gardener/etcd-backup-restore/blob/master/doc/usage/generating_ssl_certificates.md)

2. Generate `base64` strings from the certificates you generated above
   
   ```bash
   base64 -i <filepath of certificate file>
   ```

3. Add the generated `base64` strings into the secret objects in `example/etcd-secrets.yaml`

4. Apply the secrets to your kubernetes cluster
   
   ```bash
   kubectl apply -f example/etcd-secrets.yaml
   ```

#### Update the configmap

1. Uncomment `client-transport-security` from `etcd-configmap.yaml` to have a configmap that contains secrets for etcd

2. Apply the config map for etcd
   
   ```bash
   kubectl apply -f example/etcd-configmap.yaml 
   ```

#### Add TLS to the application

1. Uncomment the extra TLS flags and volume mounts that are part of `example/etcd-sts.yaml`

2. Apply the etcd statefulset
   
   ```bash
   kubectl apply -f example/etcd-sts.yaml
   ```
