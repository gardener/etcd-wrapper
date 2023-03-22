package types

import (
	"fmt"
	"net/url"
)

const (
	DefaultTLSEnabled       = false
	DefaultSideCarAddress   = "http://127.0.0.1:8080"
	DefaultExitCodeFilePath = "/Users/I544000/go/src/github.tools.sap/I062009/etcd-bootstrapper/exit_code" //"/var/etcd/data/exit_code" //TODO @aaronfern: use proper exit_code path here
)

type SidecarConfig struct {
	BaseAddress      string
	TLSEnabled       bool
	CaCertBundlePath *string
}

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
