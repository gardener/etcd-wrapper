# etcd-wrapper

<img src="logo/etcd-wrapper-logo.png" style="width:300px">

[![CI Build status](https://concourse.ci.gardener.cloud/api/v1/teams/gardener/pipelines/etcd-wrapper-main/jobs/main-head-update-job/badge)](https://concourse.ci.gardener.cloud/api/v1/teams/gardener/pipelines/etcd-wrapper-main/jobs/main-head-update-job)
[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/etcd-wrapper)](https://goreportcard.com/report/github.com/gardener/etcd-wrapper)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](LICENSE)

`etcd-wrapper` configures and starts an embedded etcd.

In gardener context, each control plane (whether it is part of a seed or a shoot) gets its own etcd cluster.
An etcd cluster is realized as a [StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/).

Each etcd member is a two container `Pod` which consists of:
* `etcd-wrapper` which is the main etcd process.
* [etcd-backup-restore](https://github.com/gardener/etcd-backup-restore) sidecar which currently provides the following
  capabilities (the list is not comprehensive):
    * etcd DB validation.
    * Scheduled DB defragmentation.
    * Snapshotting - delta and full snapshots are taken at regularly.
    * Backup - delta and full snapshots are backed in an object store if one is configured.
    * Restoration - In case of a DB corruption for a single member cluster it helps in restoring from last full
      snapshot.
    * Member control operations e.g. adding the peer `etcd` process as a learner.


`etcd-wrapper` currently depends upon `backup-restore` sidecar container for the following:
* Validation of etcd DB
* Optionally restore the etcd DB from the backup object store in case the DB is corrupt (this is only relevant in a single node etcd cluster).
* Provide user provided etcd configuration.

To learn more about `etcd-wrapper` see `/docs` directory, please find the index [here](docs/README.md).

## Future improvements
* It is currently not possible to productively use `etcd-wrapper` without its sidecar([etcd-backup-restore](https://github.com/gardener/etcd-backup-restore)).
We intend to change this in the near future to make it possible to even productively consume `etcd-wrapper` independently.
* Once we move to [3.5.7](https://github.com/etcd-io/etcd/releases/tag/v3.5.7) version of etcd, then we will also leverage [Leader change notification channel](https://github.com/etcd-io/etcd/blob/6a0bbf346256960cbbe0218d6ab13443ee93e8e3/server/etcdserver/server.go#L197-L204) 
to improve the tracking of leadership change. Currently, the sidecar periodically polls for leadership change in an etcd cluster which introduces several issues.

## Feedback and Support
We always look forward to active community engagement.
Please report bugs or suggestions on how we can enhance `etcd-wrapper` on [Github Issues](https://github.com/gardener/etcd-wrapper/issues).

## More Learning

* For more information on gardener refer to the [docs](https://github.com/gardener/gardener/tree/master/docs).
* For more information on the etcd operator that is used in gardener
  see [etcd-druid](https://github.com/gardener/etcd-druid).
