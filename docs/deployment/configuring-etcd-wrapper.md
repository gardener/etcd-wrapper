# Configuring etcd-wrapper

`etcd-wrapper` container can be invoked with the following command line flags:

## Command line flags

`start-etcd` is the main command that needs to be invoked. Following are the flags that can be passed to this command.

| Flag Name                          | Type          | Required                                                                                                                                                          | Default Value | Description                                                                                                                                                                                |
| ---------------------------------- | ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| backup-restore-tls-enabled         | bool          | No                                                                                                                                                                | false         | If this is set to true then it will look for certificates to configure a HTTP client which will use TLS to communicate to the backup-restore container                                     |
| backup-restore-host-port           | string        | No                                                                                                                                                                | :8080         | Host address and port of the backup-restore  with which this container will interact during initialization. Should be of the format <host>:<port> and ***must not*** include the protocol. |
| backup-restore-ca-cert-bundle-path | string        | Yes if `backup-restore-tls-enabled` is set to true                                                                                                                | ""            | Path of CA cert bundle (This will be used when TLS is enabled via tls-enabled flag.                                                                                                        |
| etcd-server-name                   | string        | Yes, If etcd-configuration has `client-transport-security.cert-file` and `client-transport-security.key-file` and `client-transport-security.trusted-ca-file` set | ""            | Name of the server (host) which will be used to It will be used to initialize TLS for an etcd client.                                                                                      |
| etcd-client-cert-path              | string        | Yes, If etcd-configuration has `client-transport-security.cert-file` and `client-transport-security.key-file` and `client-transport-security.trusted-ca-file` set | ""            | Path to the etcd client certificate. Usually this will be the same path where the k8s secret is mounted. It will be used to initialize TLS for an etcd client.                             |
| etcd-client-key-path               | string        | Yes, If etcd-configuration has `client-transport-security.cert-file` and `client-transport-security.key-file` and `client-transport-security.trusted-ca-file` set | ""            | Path to the etcd client key. Usually this will be the same path where the k8s secret is mounted. It will be used to initialize TLS for an etcd client.                                     |
| etcd-ready-timeout                 | time.duration | No                                                                                                                                                                | 0s            | time duration the application will wait for etcd to get ready, by default it waits forever.                                                                                                |

**Example usage**

Following example shows how to pass these as command line flags when specifying a container in a StatefulSet specification.

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    instance: etcd-main
    name: etcd
  name: etcd-main
spec:
  persistentVolumeClaimRetentionPolicy:
    whenDeleted: Retain
  replicas: 1
  selector:
    matchLabels:
      instance: etcd-main
      name: etcd
  serviceName: etcd-main-peer
  template:
    metadata:
      labels:
        instance: etcd-main
        name: etcd
    spec:
      containers:
      - args:
        - start-etcd 
        - --backup-restore-tls-enabled=true
        - --backup-restore-host-port=etcd-main-local:8080
        - --etcd-server-name=etcd-main-local
        - --etcd-client-cert-path=/var/etcd/ssl/client/client/tls.crt
        - --etcd-client-key-path=/var/etcd/ssl/client/client/tls.key
        - --backup-restore-ca-cert-bundle-path=/var/etcd/ssl/client/ca/bundle.crt
        image: etcd-wrapper:tag # change this to where you have hosted the docker image for etcd-wrapper along with its tag
        imagePullPolicy: IfNotPresent
```
