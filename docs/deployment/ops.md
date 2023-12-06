# Operations & Debugging

## Motivation

To ensure a reduced attack surface, `etcd-wrapper` docker image uses a distroless image as its base image. See [Dockerfile](../../Dockerfile). As a consequence there is not going to be any shell available in the container. This constraints the operators who would wish to debug an issue by connecting and querying the etcd process.

The purpose of this document is to demonstrate a way to interact with etcd. This provides a way to access and interact with etcd so that commands can be run on the etcd process itself to test etcd operations and as a way to help debug any issue that etcd might run into.

## Ephemeral Containers

We propose to use an [Ephemeral container](https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/) as a debug container. Since `etcd-wrapper` does not have a shell, nor does it have any etcd cli tools in-built, this ops container helps to exec into a pod and perform operations. <br />A new container is added to the pod and this container can be exec'd into. Since this container contains bash, curl, and etcdctl by default, it should be sufficient to debug etcd related issues. If other tools are required, they can easily be installed using bash or one could also enhance the image of this ephemeral container.

> This process involves building an image to be used in debugging. Since we do not have a dedicated image registry for `etcd-wrapper` yet, you will need to use your own personal image registry to host this image

### Build image

1. Use the dockerfile present in `ops/` to build the debug image
   
   ```bash
   docker build -f ops/Dockerfile -t <registry_name>/<image_name>:<image_tag> ./
   ```

### Uploading Docker Image

There are two options to upload the docker image:

1. After building and tagging the image one can simply use the following command to push the image to a public docker repository or private docker repository which your organization supports via the following command:
   
   ```bash
   docker push <registry_name>/<image_name>:<image_tag>
   ```

2. For local development it is usually much faster to just build the docker image locally (assuming that you have [docker-desktop](https://www.docker.com/products/docker-desktop/) already installed). To enable KIND cluster to pull these images execute the following command:
   
   ```bash
   # cluster-name is optional if you are using the default 'kind' cluster
   # image-name and image-tag are the same that you have used when building docker images locally
   > kind load -n <cluster-name> docker-image <image-name:image-tag>
   ```

> NOTE: If you make any changes to the Dockerfile please do not forget to build and re-load the image for kind cluster to see it.

### Using the image

> This step adds a new container to an already running pod. Make sure that an etcd pod is already running before you run this step.

```bash
> kubectl debug -it etcd-test-0 --image=<registry_name>/<repo_name>:<version> --target=etcd
```

Now you have a new container as a part of the pod. You can exec into this newly created container and freely run any bash command or any etcdctl command.

### Get ETCD PKI resource paths and common commands

If TLS has been enabled then you will need to provide paths to CA-Cert, Server-Cert and  Server-Key to connect to etcd process via `etcdctl`. To get the paths a convenience script is provided which will print all required PKI resource paths.
This script also doubles up as a cheatsheet that contains some of the most common `etcdctl` commands that an operator might use along with their PKI resource paths

```bash
> print-etcd-cheatsheet
 📌 ETCD PKI resource paths:
  --------------------------------------------------
  --cacert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/ca/bundle.crt
  --cert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.crt
  --key=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.key

 📌 ETCD configuration path:
  --------------------------------------------------
  In etcd-wrapper: proc/<etcd-wrapper-process-id>/root/home/nonroot/etcd.conf.yaml
  In etcd-backup-restore: proc/<backup-restore-process-id>/root/home/nonroot/etcd.conf.yaml

 📌 ETCD data directory:
  --------------------------------------------------
  proc/<etcd-wrapper-process-id>/root/var/etcd/data

 📌 ETCD maintenance commands:
  --------------------------------------------------
  List all etcd members:
  etcdctl member list -w table \
  --cacert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/ca/bundle.crt \
  --cert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.crt \
  --key=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.key \
  --endpoints=https://etcd-main-local:2379

  Update etcd member peer URL:
  etcdctl member update <member-id> \
  --peer-urls=<new-peer-url-to-set> \
  --cacert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/ca/bundle.crt \
  --cert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.crt \
  --key=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.key \
  --endpoints=https://etcd-main-local:2379

  Get endpoint status for the etcd cluster:
  etcdctl endpoint -w table --cluster status \
  --cacert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/ca/bundle.crt \
  --cert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.crt \
  --key=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.key \
  --endpoints=https://etcd-main-local:2379

  List all alarms:
  etcdctl alarm list \
  --cacert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/ca/bundle.crt \
  --cert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.crt \
  --key=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.key \
  --endpoints=https://etcd-main-local:2379

  Disarm all alarms:
  etcdctl alarm disarm \
  --cacert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/ca/bundle.crt \
  --cert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.crt \
  --key=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.key \
  --endpoints=https://etcd-main-local:2379

  Defragment etcd:
  etcdctl defrag \
  --cacert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/ca/bundle.crt \
  --cert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.crt \
  --key=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.key \
  --endpoints=https://etcd-main-local:2379

  Change leadership:
  etcdctl move-leader <new-leader-member-id> \
  --cacert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/ca/bundle.crt \
  --cert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.crt \
  --key=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.key \
  --endpoints=https://etcd-main-local:2379

 📌 ETCD Key-Value commands:
  --------------------------------------------------

  Get key details:
  etcdctl get <key> \
  --cacert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/ca/bundle.crt \
  --cert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.crt \
  --key=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.key \
  --endpoints=https://etcd-main-local:2379

  Get only value for a given key:
  etcdctl get <key> --print-value-only \
  --cacert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/ca/bundle.crt \
  --cert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.crt \
  --key=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.key \
  --endpoints=https://etcd-main-local:2379

  List all keys:
  etcdctl get "" --prefix --keys-only \
  --cacert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/ca/bundle.crt \
  --cert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.crt \
  --key=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.key \
  --endpoints=https://etcd-main-local:2379

  Put a value against a key:
  etcdctl put <key> <value> \
  --cacert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/ca/bundle.crt \
  --cert=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.crt \
  --key=proc/<etcd-wrapper-process-id>/root/var/etcd/ssl/client/client/tls.key \
  --endpoints=https://etcd-main-local:2379
```

### Work directory

Ephemeral container is started with a non-root user (65532), which does not provide write access to any existing directory. For this reason we have created a `work` directory which is owned by non-root (65532) user. You can use this directory for any temporary creation/copy of files.