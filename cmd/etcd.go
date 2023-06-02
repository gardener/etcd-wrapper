// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		UsageLine: "",
		LongDesc: `Initializes the etcd data directory by coordinating with a backup-sidecar container
and starts an embedded etcd.

Flags:
	--backup-restore-tls-enabled
		Enables TLS for communicating with backup-restore if its value is true. It is disabled by default.
	--backup-restore-host-port string
		Host address and port of the backup restore with which this container will interact during initialization. Should be of the format <host>:<port> and must not include the protocol.
	--backup-restore-ca-cert-bundle-path string
		Path of CA cert bundle (This will be used when TLS is enabled via backup-restore-tls-enabled flag.
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
	fs.BoolVar(&config.BackupRestore.TLSEnabled, "backup-restore-tls-enabled", types.DefaultBackupRestoreTLSEnabled, "Enables TLS for communicating with backup-restore container")
	fs.StringVar(&config.BackupRestore.HostPort, "backup-restore-host-port", types.DefaultBackupRestoreHostPort, "Host and Port to be used to connect to the backup-restore container")
	fs.StringVar(&config.BackupRestore.CaCertBundlePath, "backup-restore-ca-cert-bundle-path", "", "File path of CA cert bundle to help establish TLS communication with backup-restore container")
	fs.StringVar(&config.EtcdClientTLS.ServerName, "etcd-server-name", "", "Name of the server (host) which will be used to configure TLS config to connect to the etcd server process")
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
