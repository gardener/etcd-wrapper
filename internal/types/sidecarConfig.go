package types

import (
	"fmt"
	"strings"
)

const (
	// DefaultTLSEnabled defines the default TLS state of the application
	DefaultTLSEnabled = false
	// DefaultSideCarHostPort defines the default sidecar host and port
	DefaultSideCarHostPort = ":8080"
	// DefaultExitCodeFilePath defines the default file path for the file that stores the exit code of the previous run
	DefaultExitCodeFilePath = "/Users/I544000/go/src/github.tools.sap/I062009/etcd-bootstrapper/exit_code" //"/var/etcd/data/exit_code" //TODO @aaronfern: use proper exit_code path here
)

// SidecarConfig defines parameters needed for the sidecar
type SidecarConfig struct {
	HostPort         string
	TLSEnabled       bool
	CaCertBundlePath *string
}

// Validate validates all parameters passed as sidecar configuration
func (c *SidecarConfig) Validate() error {
	if strings.HasPrefix(c.HostPort, "http:") || strings.HasPrefix(c.HostPort, "https:") {
		return fmt.Errorf("sidecar-host-port should not contain scheme")
	}
	if c.TLSEnabled {
		if *c.CaCertBundlePath == "" {
			return fmt.Errorf("certificate bundle path cannot be empty when TLS is enabled")
		}
	}
	return nil
}
