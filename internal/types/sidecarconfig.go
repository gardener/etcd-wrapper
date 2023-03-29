package types

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// DefaultTLSEnabled defines the default TLS state of the application
	DefaultTLSEnabled = false
	// DefaultSideCarHostPort defines the default sidecar host and port
	DefaultSideCarHostPort = ":8080"
	// DefaultExitCodeFilePath defines the default file path for the file that stores the exit code of the previous run
	DefaultExitCodeFilePath = "/var/etcd/data/exit_code"
)

// SidecarConfig defines parameters needed for the sidecar
type SidecarConfig struct {
	HostPort         string
	TLSEnabled       bool
	CaCertBundlePath *string
}

// Validate validates all parameters passed as sidecar configuration
func (c *SidecarConfig) Validate() (err error) {
	splits := strings.Split(c.HostPort, ":")
	if len(splits) < 2 {
		err = errors.Join(err, fmt.Errorf("both host and port needs to be specified and should be adhere to format: <host>:<port>"))
	}

	if strings.HasPrefix(c.HostPort, "http:") || strings.HasPrefix(c.HostPort, "https:") {
		err = errors.Join(err, fmt.Errorf("sidecar-host-port should not contain scheme"))
	}
	if c.TLSEnabled {
		if c.CaCertBundlePath == nil || strings.TrimSpace(*c.CaCertBundlePath) == "" {
			err = errors.Join(err, fmt.Errorf("certificate bundle path cannot be nil or empty when TLS is enabled"))
		}
	}
	return
}

// GetBaseAddress returns the complete address of the backup restore sidecar
func (c *SidecarConfig) GetBaseAddress() string {
	scheme := SchemeHTTP
	if c.TLSEnabled {
		scheme = SchemeHTTPS
	}
	return fmt.Sprintf("%s://%s", scheme, c.HostPort)
}
