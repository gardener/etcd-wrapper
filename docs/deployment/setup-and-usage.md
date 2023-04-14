# Configuring etcd-wrapper

### Command line arguments
| Flag Name | Type | Required | Default Value | Description |
| --- | --- | --- | --- | --- |
| tls-enabled | bool | No | false | Enables TLS for the application |
| sidecar-host-port | string | No | :8080 | Host address and port of the backup restore sidecar with which this container will interact during initialization. Should be of the format <host>:<port> and must not include the protocol. |
| sidecar-ca-cert-bundle-path | string | No | "" | Path of CA cert bundle (This will be used when TLS is enabled via tls-enabled flag. |
| etcd-wait-ready-timeout | time.duration | No | 0s | time duration the application will wait for etcd to get ready, by default it waits forever. |

## Running etcd-wrapper locally
etcd-wrapper has the capability of running locally on your host machine as a process

1. Clone the repository locally using the following command
<details>
<summary>Prerequisites</summary>
<br>
Install Git

On MacOS run:
```bash
brew install git
```
For other OS or for more detailed instructions please follow the [installation instructions](https://git-scm.com/downloads).
</details>

```bash

 ```

## Running etcd-wrapper as a container on a cluster
etcd-wrapper can be run as a container on a cluster. It is designed to run in tandem with [etcd-backup-restore](https://github.com/gardener/etcd-backup-restore) as a sidecar container within a single pod.

To try out etcd-wrapper along with etcd-backup-restore, we have ready-made objects that you can just apply to a running kubernetes cluster and have an etcd pod up and running.

*Note: we do not have a registry to host etcd-wrapper images yet (don't worry, we will soon, and we will update this doc as soon as that happens), so you will have to build an image and use your own personal docker registry to host that image*

<details>
<summary>Prerequisites</summary>

#### Install Git

On MacOS run:
```bash
brew install git
```
For other OS or for more detailed instructions please follow the [installation instructions](https://git-scm.com/downloads).
<br />
#### Install Docker

In order to test etcd-wrapper containers you will need a local kubernetes setup. Easiest way is to first install Docker. This becomes a pre-requisite to setting up either a vanilla KIND/minikube cluster or a local Gardener cluster.

On MacOS run:
```bash
brew install -cash docker
```
For other OS, follow the [installation instructions](https://docs.docker.com/get-docker/).

#### Installing Kubectl

To interact with the local Kubernetes cluster you will need kubectl. On MacOS run:
```bash
brew install kubernetes-cli
```
For other OS, follow the [installation instructions](https://kubernetes.io/docs/tasks/tools/install-kubectl/).
</details>

### Clone the repository
```bash
git clone https://github.com/gardener/etcd-wrapper.git
```

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
1. Apply prerequisites
```bash
kubectl apply -f docs/deployment/prerequisites.yaml 
```
2. Apply etcd sts
```bash
kubectl apply -f docs/deployment/etcd-sts.yaml 
```
*Before you apply the above yaml, make sure to specify the image you built on line no 27. If you don't want to build an image yourself, an image is already specified in the yaml that can be used*
