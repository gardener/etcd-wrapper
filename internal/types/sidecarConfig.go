package types

import (
	"fmt"
	"net/url"
)

const (
	// DefaultTLSEnabled defines the default TLS state of the application
	DefaultTLSEnabled = false
	// DefaultSideCarAddress defines the default sidecar base address
	DefaultSideCarAddress = "http://127.0.0.1:8080"
	// DefaultExitCodeFilePath defines the default file path for the file that stores the exit code of the previous run
	DefaultExitCodeFilePath = "/Users/I544000/go/src/github.tools.sap/I062009/etcd-bootstrapper/exit_code" //"/var/etcd/data/exit_code" //TODO @aaronfern: use proper exit_code path here
)

// SidecarConfig defines parameters needed for the sidecar
type SidecarConfig struct {
	BaseAddress      string
	TLSEnabled       bool
	CaCertBundlePath *string
}

// Validate validates all parameters passed as sidecar configuration
func (c *SidecarConfig) Validate() error {
	baseURL, err := url.Parse(c.BaseAddress)
	if err != nil {
		return fmt.Errorf("error parsing base address URL: %s: %v", c.BaseAddress, err)
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return fmt.Errorf("invalid url passed as sidecar base address %s", err.Error())
	}
	if c.TLSEnabled {
		if *c.CaCertBundlePath == "" {
			return fmt.Errorf("certificate bundle path cannot be empty when TLS is enabled")
		}
		if baseURL.Scheme != "https" {
			return fmt.Errorf("incorrect scheme for sidecar address. https URL expected")
		}
	}
	return nil
}
