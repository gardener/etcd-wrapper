# Local KIND Cluster based ETCD Cluster Setup

To make development and testing of `etcd-wrapper` easier we also provide [scripts](../../hack/local-dev) which will help you generate and deploy all required resources to a locally started [KIND](https://kind.sigs.k8s.io/) cluster.

## Setup KIND cluster

To setup a KIND cluster use the following script:

> assuming that you are in the root directory of the etcd-wrapper project

```bash
> ./hack/local-dev/kind.sh -h
usage: kind.sh [Options]
  Options:
   -n | --cluster-name  <kind-cluster-name>         (Optional) name of the kind cluster. if not specified, uses default name 'kind'
   -f | --force-create                              If this flag is specified then it will always create a fresh cluster.
   -g | --feature-gates <feature gates to enable>   Comma separated list of feature gates that needs to be enabled.
   -d | --delete                                    Deletes a kind cluster. If a name is provided via '-n | --cluster-name' then it will delete that cluster else it deletes the default kind cluster with name 'kind'. If this option is not used then it will by default create a kind cluster.
```

**Example Usage:**

```bash
> ./hack/local-dev/kind.sh -n wrapper-test
```

## Uploading Docker Images

The setup will create a `Pod` with two containers: `etcd-wrapper` and `etcd-backup-restore`. If you are building images for these projects locally then make sure that you upload these images for KIND to pull them without problems. There are two options to upload docker images:

1. Build the docker image, tag it and push the image to a public repository or a private repository which your organization supports.

2. For local development it is usually much faster to just build the docker image locally (assuming that you have [docker-desktop](https://www.docker.com/products/docker-desktop/) already installed). To enable KIND cluster to pull these images execute the following command:
   
   ```bash
   # cluster-name is optional if you are using the default 'kind' cluster
   # image-name and image-tag are the same that you have used when building docker images locally
   > kind load -n <cluster-name> docker-image <image-name:image-tag>
   ```

## Pre-requisites

Scripts that are created to create resources for an etcd cluster uses the following tools. Please ensure that you have installed them. The script will also point out in case a tool used is missing with an appropriate link which has the installation instructions.

```bash
# YQ is a YAML processor, please see install instructions at 
# https://github.com/mikefarah/yq#install
> yq
```

```bash
# CFSSL is a tool that is used to generate PKI resources
# To install follow install instructions at https://github.com/cloudflare/cfssl#installation
> cfssl
```

```bash
# Skaffold eases the setup of k8s resources and offers dev and debug capabilities
# To install please see install instructions at https://skaffold.dev/docs/install/
> skaffold
```

## Setup ETCD Cluster

To setup an ETCD cluster use the following script:

> assuming that you have proper KUBECONFIG already exported and your terminal is currently pointing to the correct context.

```bash
> ./hack/local-dev/etcd-up.sh -h
 usage: etcd-up.sh [Options]
  Options:
   -n | --namespace                   <namespace>                           (Optional) kubernetes namespace where etcd resources will be created. if not specified uses 'default'
   -s | --cluster-size                <size of etcd cluster>                (Optional) size of an etcd cluster. Supported values are 1 or 3. Defaults to 1
   -t | --tls-enabled                 <is-tls-enabled>                      (Optional) controls the TLS communication amongst peers and between etcd and its client.Possible values: ['true' | 'false']. Defaults to 'false'
   -i | --etcd-instance-name          <name of etcd instance>               (Option) name of the etcd instance. Defaults to 'etcd-main'
   -e | --cert-expiry                 <certificate expiry>                  (Optional) common expiry for all certificates generated. Defaults to '12h'
   -m | --etcd-br-image               <image:tag of etcd-br container>      (Required) Image (with tag) for etcdbr container
   -w | --etcd-wrapper-image          <image of etcd-wrapper container>     (Optional) Image (without tag) for etcd-wrapper container. Skaffold will add git-commit as the tag when it builds the etcd-wrapper image.
   -r | --skaffold-run-mode           <skaffold run or debug>               (Optional) Possible values: 'run' | 'debug'. Defaults to 'run'. Will only be effective if '-d | --dry-run' is not specified.
   -f | --force-create-pki-resources                                        (Optional) If specified then it will re-create all PKI resources.
   -d | --dry-run                                                           (Optional) If set it will only generate all manifests and configuration files. The user needs to explicitly run skaffold to deploy the k8s resources.
```

**Example Usage**

```bash
./hack/local-dev/etcd-up.sh -n test-ns -s 3 -t true -i etcd-main -m etcdbr:dev -w etcd-wrapper:dev
```

The above command will eventually generate:

1. All required kubernetes manifest YAML files.

2. A [skaffold](https://skaffold.dev/) YAML which will be placed at the root of the project folder. 

`skaffold` is invoked to setup the cluster using the generated configuration YAML file.

> **NOTE:** If you set `-d | --dry-run` flag then it will only generate the resources. The user will have to explicitly create the target namespace and also run `skaffold run/dev` command to deploy the resources. If `-d | --dry-run` is not set then the script will create the target namespace and will also invoke skaffold with its passed in/default `run-mode`.

> For developement use case `skaffold` offers `dev` run-mode which you can specify via `skaffold-run-mode`. If it is specified then any change to `etcd-wrapper` will automatically build the image and the change will be re-deployed to the KIND cluster. DEV mode will only work for `etcd-wrapper` and not for any `etcd-backup-restore` development image.

### Details on resources created

The above command will create the following resources:

1. Since TLS has been enabled it will create the following PKI resources
   
   1. CA certificate and private key for etcd
   
   2. A separate CA certificate and private key for peer communication
   
   3. Certificate and private key for etcd server
   
   4. Certificate and private key for etcd client
   
   5. Certificate and private key for etcd peers
   
   You can access these resources at `/hack/local-dev/secrets`

2. It will use the above PKI resources to create k8s secret manifests and put them in `hack/local-dev/manifests/common` along with other common k8s manifests which will always be produced irrespective of whether the etcd cluster is `single-node` or `multi-node`

3. For a multi-node etcd cluster, k8s manifests specific to multi-node variant will be put in `/hack/local-dev/manifests/multi-node`.

4. For a single-node etcd cluster, k8s manifests specific to single-node variant will be put in `/hack/local-dev/manifests/single-node`

5. A skaffold configuration file is also generated and put at the root of the project.

> **NOTE:** Once the PKI resources are generated using the default expiry of 12h or a user provided expiry via `-e | --cert-expiry` flag, these will not be re-generated every time you run `./hack/local-dev/etcd-up.sh` unless you explicitly specify `-f | --force-create-pki-resources` flag.

## ## Bringing down ETCD cluster

Bringing down an ETCD cluster will delete all the k8s resources that are created for the etcd-cluster. To delete an ETCD cluster use the following script:

```bash
./hack/local-dev/etcd-down.sh -h 
 usage: etcd-down.sh [Options]
  Options:
   -n | --namespace <namespace> kubernetes namespace where etcd resources are created. If not specified uses 'default'
```

**Example Usage:**

```bash
./hack/local-dev/etcd-down.sh --namespace test-ns
```

## Cleaning up all generated files

If you wish to clean up all generated files (PKI resources and k8s manifests) then use the following script:

```bash
./hack/local-dev/cleanup.sh
```

> **WARNING:** If you cleanup all generated files then you will not be able to use `./hack/local-dev/etcd-down.sh` as the cleaup will also remove the generated skaffold.yaml which is used to remove all etcd resources from the target k8s cluster.

## Bringing down KIND cluster

To delete the KIND cluster use the following script:

```bash
./hack/local-dev/kind.sh -n wrapper-test -d
```
