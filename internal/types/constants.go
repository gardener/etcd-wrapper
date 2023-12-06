// Copyright 2023 SAP SE or an SAP affiliate company
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
