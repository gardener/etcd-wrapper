package util

import (
	"crypto/x509"
	"os"
)

func CreateCACertPool(caCertBundlePath string) (*x509.CertPool, error) {
	caCertBundle, err := os.ReadFile(caCertBundlePath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertBundle)
	return caCertPool, nil
}
