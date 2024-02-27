// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package types

import "go.uber.org/zap/zapcore"

const (
	// DefaultBackupRestoreTLSEnabled defines the default TLS state of the application
	DefaultBackupRestoreTLSEnabled = false
	// DefaultBackupRestoreHostPort defines the default sidecar host and port
	DefaultBackupRestoreHostPort = ":8080"
	// DefaultExitCodeFilePath defines the default file path for the file that stores the exit code of the previous run
	DefaultExitCodeFilePath = "/var/etcd/data/exit_code"
	// DefaultLogLevel defines the default log level for any zap loggers created
	DefaultLogLevel = zapcore.InfoLevel
)
