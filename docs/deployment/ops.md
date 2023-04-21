# Operations & Debugging

## Motivation

To ensure a reduced attack surface, `etcd-wrapper` docker image uses a distroless image as its base image. See [Dockerfile](../../Dockerfile). As a consequence there is not going to be any shell available in the container. This constraints the operators who would wish to debug an issue by connecting and querying the etcd process.

The purpose of this document is to demonstrate a way to interact with etcd. This provides a way to access and interact with etcd so that commands can be run on the etcd process itself to test etcd operations and as a way to help debug any issue that etcd might run into.

## Ephemeral Containers

We propose to use an [Ephemeral container](https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/)s as a debug container. Since `etcd-wrapper` does not have a shell, nor does it have any etcd cli tools in-built, this ops container helps to exec into a pod and perform operations. <br />A new container is added to the pod and this container can be exec'd into. Since this container container bash, curl, and etcdctl by default, it should be sufficient to debug etcd related issues. If other tools are required, they can easily be installed using bash or one could also enhance the image of this ephemeral container.

> This process involves building an image to be used in debugging. Since we do not have a dedicated image registry for `etcd-wrapper` yet, you will need to use your own personal image registry to host this image

### Build image

1. Use the dockerfile present in `ops/` to build the debug image
   
   ```bash
   docker build -f ops/Dockerfile -f <registry_name>/<repo_name>:<version> ./
   ```

2. Push the image to image registry
   
   ```bash
   docker push <registry_name>/<repo_name>:<version> 
   ```

### Using the image

> This step adds a new container to an already running pod. Make sure that an etcd pod is already running before you run this step.

```bash
kubectl debug -it etcd-test-0 --image=<registry_name>/<repo_name>:<version> --target=etcd
```

Now you have a new container as a part of the pod. You can exec into this newly created container and freely run any bash command or any etcdctl command.