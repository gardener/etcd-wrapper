// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"flag"
	"time"

	"github.com/gardener/etcd-wrapper/internal/types"

	"github.com/gardener/etcd-wrapper/internal/app"
	"go.uber.org/zap"
)

var (
	// EtcdCmd initializes and starts an embedded etcd.
	EtcdCmd = Command{
		Name:      "start-etcd",
		ShortDesc: "Starts the etcd-wrapper application by initializing and starting an embedded etcd",
		LongDesc: `Initializes the etcd data directory by coordinating with a backup-sidecar container
and starts an embedded etcd which is by default exposed on port 2379 for client traffic.

Flags:
	--etcd-wrapper-port
		Port used by etcd-wrapper to expose the server. Default: 9095
	--backup-restore-tls-enabled
		Enables TLS for communicating with backup-restore if its value is true. It is disabled by default.
	--backup-restore-host-port
		Host address and port of the backup restore with which this container will interact during initialization. Should be of the format <host>:<port> and must not include the protocol.
	--backup-restore-ca-cert-bundle-path
		Path of CA cert bundle (This will be used when TLS is enabled via backup-restore-tls-enabled flag.
    --etcd-client-port
		Client port when talking to etcd. Default: 2379
    --etcd-client-cert-path
		Path of TLS certificate of the etcd client (This will be used if client-transport-security is set in the etcd configuration).
	--etcd-client-key-path
		Path of TLS key of the etcd client (This will be used if client-transport-security is set in the etcd configuration).
	--etcd-server-name
		Name of the server (host) which will be used to configure TLS config to connect to the etcd server process.
	--etcd-ready-timeout
		time duration the application will wait for etcd to get ready, by default it waits forever.`,
		AddFlags: AddEtcdFlags,
		Run:      InitAndStartEtcd,
	}
	config = types.Config{}
	// etcdReadyTimeout is the timeout for an embedded etcd server to be ready.
	etcdReadyTimeout time.Duration
)

// AddEtcdFlags adds flags from the parsed FlagSet into application structs
func AddEtcdFlags(fs *flag.FlagSet) {
	fs.IntVar(&config.EtcdWrapperPort, "etcd-wrapper-port", 9095, "Port used by etcd-wrapper to expose the server. Default: 9095")
	fs.BoolVar(&config.BackupRestore.TLSEnabled, "backup-restore-tls-enabled", types.DefaultBackupRestoreTLSEnabled, "Enables TLS for communicating with backup-restore container")
	fs.StringVar(&config.BackupRestore.HostPort, "backup-restore-host-port", types.DefaultBackupRestoreHostPort, "Host and Port to be used to connect to the backup-restore container")
	fs.StringVar(&config.BackupRestore.CaCertBundlePath, "backup-restore-ca-cert-bundle-path", "", "File path of CA cert bundle to help establish TLS communication with backup-restore container")
	fs.StringVar(&config.EtcdClientTLS.ServerName, "etcd-server-name", "", "Name of the server (host) which will be used to configure TLS config to connect to the etcd server process")
	fs.IntVar(&config.EtcdClientPort, "etcd-client-port", 2379, "Client port when talking to etcd. Default: 2379")
	fs.StringVar(&config.EtcdClientTLS.CertPath, "etcd-client-cert-path", "", "File path of ETCD client certificate to help establish TLS communication of the client to ETCD")
	fs.StringVar(&config.EtcdClientTLS.KeyPath, "etcd-client-key-path", "", "File path of ETCD client key to help establish TLS communication of the client to ETCD")
	fs.DurationVar(&etcdReadyTimeout, "etcd-ready-timeout", 0, "Time duration to wait for etcd to be ready")
}

// InitAndStartEtcd sets up and starts an embedded etcd
func InitAndStartEtcd(ctx context.Context, cancelFn context.CancelFunc, logger *zap.Logger) error {
	etcdApp, err := app.NewApplication(ctx, cancelFn, config, etcdReadyTimeout, logger)
	if err != nil {
		return err
	}
	if err := etcdApp.Setup(); err != nil {
		return err
	}
	return etcdApp.Start()
}
