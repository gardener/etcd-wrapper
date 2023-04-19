# Etcd bootstrap

## Overview
The purpose of this document is to explain the process of how etcd-wrapper starts up. We call this process bootstrapping.

This is the description of the steps and checks that have to be done in order for etcd-wrapper to properly start up. Etcd-wrapper is not designed to run as a stand-alone application, but in tandem with [etcd-backup-restore](https://github.com/gardener/etcd-backup-restore) - as a sidecar. This sidecar performs operations to support etcd such as data directory initialization and restoration, regular data directory defragmentation, automatic backups, etc.

## Lifecycle
The following is the detailed lifecycle of etcd-wrapper. The workings of etcd-backup-restore are not explained in detailed and are mentioned barely enough to explain etcd-wrapper lifecycle. Please refer to etcd-backup-restore's [github](https://github.com/gardener/etcd-backup-restore) page for more details

1. After validating the command and options, an `initialization` loop is started
   1. Check the `exit_code` to see if the previous run (if any) terminated gracefully of not
   2. Instruct etcd-backup-restore to perform a data directory validation
      > Graceful termination of the previous run leads to sanity validation, else a full validation <br />For more details about the validations modes check [here](https://github.com/gardener/etcd-backup-restore/blob/master/doc/proposals/validation.md)
   3. Periodically probe data directory validation status and continue probing until a `Success` status is returned
      > The actual validation logic is part of `etcd-backup-restore` and is out of scope of this document. Check [here](https://github.com/gardener/etcd-backup-restore/blob/master/doc/proposals/validation.md) for more details. <br /> Depending on the outcome of data directory validation, data restoration may be required if data is found to be corrupt
2. Query the `/config` endpoint of `etcd-backup-restore` to fetch the etcd application configuration
3. Set up a readiness probe at `/readyz` where anyone can query to verify if the etcd application is running
4. Start an embedded etcd using the fetched configuration
5. Delete the `exit_code` file and block until the application context is cancelled or until the etcd server stops or returns an error
6. On application termination write the code of the termnation signal received into a special file `exit_code` to be used the next time the application runs


[//]: # (TODO: Add diag)