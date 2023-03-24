package types

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
)

const (
	defaultTestHostPort         = "locahost:2379"
	defaultTestCaCertBundlePath = "/var/etcd/ssl/client/ca/cabundle.crt"
)

func TestGetBaseAddressWithTLSEnabled(t *testing.T) {
	g := NewWithT(t)
	config := createSidecarConfig(true, defaultTestHostPort)
	expectedBaseAddress := fmt.Sprintf("https://%s", config.HostPort)
	g.Expect(config.GetBaseAddress()).To(Equal(expectedBaseAddress))
}

func TestGetBaseAddressWithTLSDisabled(t *testing.T) {
	g := NewWithT(t)
	config := createSidecarConfig(false, defaultTestHostPort)
	expectedBaseAddress := fmt.Sprintf("http://%s", config.HostPort)
	g.Expect(config.GetBaseAddress()).To(Equal(expectedBaseAddress))
}

func TestValidate(t *testing.T) {
	emptyCaCertBundlePath := ""
	table := []struct {
		description      string
		tlsEnabled       bool
		hostPort         string
		caCertBundlePath *string
		expectedError    bool
	}{
		{"missing host should result in error", false, "2379", nil, true},
		{"missing port should result in error", false, "localhost", nil, true},
		{"should allow empty host", false, ":2379", nil, false},
		{"should disallow specifying scheme", false, "http://localhost:2379", nil, true},
		{"should disallow nil caCertBundlePath when TLS is enabled", true, ":2379", nil, true},
		{"should disallow empty caCertBundlePath when TLS is enabled", true, ":2379", &emptyCaCertBundlePath, true},
	}
	for _, entry := range table {
		g := NewWithT(t)
		t.Log(entry.description)
		c := createSidecarConfig(entry.tlsEnabled, entry.hostPort)
		c.CaCertBundlePath = entry.caCertBundlePath
		err := c.Validate()
		g.Expect(err != nil).To(Equal(entry.expectedError))
	}
}

func createSidecarConfig(tlsEnabled bool, hostPort string) SidecarConfig {
	var caCertBundlePath string
	if tlsEnabled {
		caCertBundlePath = defaultTestCaCertBundlePath
	}
	return SidecarConfig{
		HostPort:         hostPort,
		TLSEnabled:       tlsEnabled,
		CaCertBundlePath: &caCertBundlePath,
	}
}
