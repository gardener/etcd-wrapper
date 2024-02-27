// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gardener/etcd-wrapper/internal/util"
)

// Config holds the application configuration for etcd-wrapper.
type Config struct {
	// BackupRestore is the configuration to interact with the backup-restore container.
	BackupRestore BackupRestoreConfig
	// EtcdClientTLS is the TLS configuration required to configure a client when TLS is enabled when interacting with the embedded etcd.
	EtcdClientTLS EtcdClientTLSConfig
}

// EtcdClientTLSConfig holds the TLS configuration to configure a etcd client.
type EtcdClientTLSConfig struct {
	// ServerName is the name of the etcd server. It should be ensured that the name used
	// should also be specified as one of the name(s) in the subject-alternate names in the etcd server certificate.
	ServerName string
	// CertPath is the path to the client certificate
	CertPath string
	// KeyPath is the path to the client key
	KeyPath string
}

// BackupRestoreConfig defines parameters needed to interact with the backup-restore container
type BackupRestoreConfig struct {
	HostPort         string
	TLSEnabled       bool
	CaCertBundlePath string
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
		if strings.TrimSpace(c.CaCertBundlePath) == "" {
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
