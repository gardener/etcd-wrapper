# Configuring etcd-wrapper

### Command line arguments
| Flag Name | Type | Required | Default Value | Description |
| --- | --- | --- | --- | --- |
| tls-enabled | bool | No | false | Enables TLS for the application |
| sidecar-host-port | string | No | :8080 | Host address and port of the backup restore sidecar with which this container will interact during initialization. Should be of the format <host>:<port> and must not include the protocol. |
| sidecar-ca-cert-bundle-path | string | No | "" | Path of CA cert bundle (This will be used when TLS is enabled via tls-enabled flag. |
| etcd-wait-ready-timeout | time.duration | No | 0s | time duration the application will wait for etcd to get ready, by default it waits forever. |

## Running etcd-wrapper locally
> Right now etcd-wrapper does not have support to run locally as a process. However, we are working on this and will have this feature in soon

## Running etcd-wrapper in a local kind cluster
etcd-wrapper can be run as a container on a cluster. It has a dependency and is designed to run in tandem with [etcd-backup-restore](https://github.com/gardener/etcd-backup-restore) as a sidecar container within a single pod.

To try out etcd-wrapper along with etcd-backup-restore, we have ready-made objects that you can apply to a running kubernetes cluster and have an etcd pod up and running.

In this section, we show how to run etcd-wrapper on a local _kind_ cluster. These steps can however be replicated as is on any other kubernetes cluster without modification
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
    ```
      docker push <registry_name>/<repo_name>:<version>
    ```

### Run application
1. Apply role, rolebinding, and service account
    > etcd-backup-restore needs permissions to perform opertions on certain objects
    ```bash
      kubectl apply -f example/role.yaml
      kubectl apply -f example/rolebinding.yaml
      kubectl apply -f example/serviceaccount.yaml 
    ```
2. Apply etcd member lease
    > etcd-backup-restore needs the member lease to push regular heartbeats 
    ```bash
      kubectl apply -f example/member-lease.yaml
    ```
3. Apply client service
    > The client service is used as an endpoint by any client to send requests to etcd
    ```bash
      kubectl apply -f example/client-svc.yaml
    ```
4. Apply etcd configuration
    > all configuration required by the etcd process to start is stored in this configmap
    ```bash
      kubectl apply -f example/etcd-configmap.yaml
    ```
5. Apply etcd statefulset
    ```bash
      kubectl apply -f docs/deployment/etcd-sts.yaml 
    ```

### Cleanup
When you are done, you can clean up the entire setup by deleting the kind cluster
```bash
kind delete cluster
```