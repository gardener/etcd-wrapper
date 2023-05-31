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

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gardener/etcd-wrapper/internal/util"
)

// BackupRestoreConfig defines parameters needed for the sidecar
type BackupRestoreConfig struct {
	HostPort         string
	TLSEnabled       bool
	CaCertBundlePath *string
}

// Validate validates backup-restore configuration.
func (c *BackupRestoreConfig) Validate() (err error) {
	splits := strings.Split(c.HostPort, ":")
	if len(splits) < 2 {
		err = errors.Join(err, fmt.Errorf("both host and port needs to be specified and should be adhere to format: <host>:<port>"))
	}

	if strings.HasPrefix(c.HostPort, "http:") || strings.HasPrefix(c.HostPort, "https:") {
		err = errors.Join(err, fmt.Errorf("backup-restore-host-port should not contain scheme"))
	}
	if c.TLSEnabled {
		if c.CaCertBundlePath == nil || strings.TrimSpace(*c.CaCertBundlePath) == "" {
			err = errors.Join(err, fmt.Errorf("certificate bundle path cannot be nil or empty when TLS is enabled"))
		}
	}
	return
}

// GetBaseAddress returns the complete address of the backup restore container.
func (c *BackupRestoreConfig) GetBaseAddress() string {
	return util.ConstructBaseAddress(c.TLSEnabled, c.HostPort)
}

// GetHost extracts the backup-restore server host from host-port string.
func (c *BackupRestoreConfig) GetHost() string {
	host := "localhost"
	splits := strings.Split(c.HostPort, ":")
	if len(strings.TrimSpace(splits[0])) > 0 {
		host = splits[0]
	}
	return host
}
