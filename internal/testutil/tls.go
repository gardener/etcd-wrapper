// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// TLSResourceCreator is a creator for TLS resources
type TLSResourceCreator struct {
	caTemplate   *x509.Certificate
	caCert       *x509.Certificate
	caPrivateKey *rsa.PrivateKey
}

// NewTLSResourceCreator creates a new instance of TLSResourceCreator
func NewTLSResourceCreator() (*TLSResourceCreator, error) {
	caTemplate, err := createCACertTemplate()
	if err != nil {
		return nil, err
	}
	return &TLSResourceCreator{caTemplate: caTemplate}, nil
}

// CertKeyPair holds a pair certificate bytes and its corresponding rsa.PrivateKey.
type CertKeyPair struct {
	CertBytes  []byte
	PrivateKey rsa.PrivateKey
}

func pemEncode(bytes []byte, blockType string) []byte {
	block := pem.Block{
		Type:  blockType,
		Bytes: bytes,
	}
	return pem.EncodeToMemory(&block)
}

// EncodeAndWrite encodes the certificates and private key in PEM format and writes it to the provided directory.
func (c *CertKeyPair) EncodeAndWrite(dir string, certFileName, keyFileName string) error {
	key := x509.MarshalPKCS1PrivateKey(&c.PrivateKey)
	pemKeyBytes := pemEncode(key, "RSA PRIVATE KEY")
	privateKeyPath := filepath.Join(dir, keyFileName)
	err := os.WriteFile(privateKeyPath, pemKeyBytes, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write private key to dir: %s: err: %v", dir, err)
	}
	pemCertBytes := pemEncode(c.CertBytes, "CERTIFICATE")
	certPath := filepath.Join(dir, certFileName)
	err = os.WriteFile(certPath, pemCertBytes, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write certificate to dir: %s: err: %v", dir, err)
	}
	return nil
}

// CreateCACertAndKey creates a CA certificate and its private key.
func (t *TLSResourceCreator) CreateCACertAndKey() (*CertKeyPair, error) {
	// create private and public key-pair
	caPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// create the CA certificate
	caCertBytes, err := x509.CreateCertificate(rand.Reader, t.caTemplate, t.caTemplate, caPrivateKey.Public(), caPrivateKey)
	if err != nil {
		return nil, err
	}
	caCert, err := x509.ParseCertificate(caCertBytes)
	if err != nil {
		return nil, err
	}
	// Set the CA cert and private key to be used for signing client/server certificates
	t.caPrivateKey = caPrivateKey
	t.caCert = caCert

	return &CertKeyPair{
		CertBytes:  caCertBytes,
		PrivateKey: *caPrivateKey,
	}, nil
}

// CreateETCDClientCertAndKey creates a ETCD client certificate and its private key.
func (t *TLSResourceCreator) CreateETCDClientCertAndKey() (*CertKeyPair, error) {
	clientCertTemplate, err := createCertTemplate("etcd-client")
	if err != nil {
		return nil, err
	}
	clientCertTemplate.KeyUsage = x509.KeyUsageDigitalSignature
	clientCertTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}

	clientPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	clientCertBytes, err := x509.CreateCertificate(rand.Reader, clientCertTemplate, t.caCert, clientPrivateKey.Public(), t.caPrivateKey)
	if err != nil {
		return nil, err
	}
	return &CertKeyPair{
		CertBytes:  clientCertBytes,
		PrivateKey: *clientPrivateKey,
	}, nil
}

func createCACertTemplate() (*x509.Certificate, error) {
	caTemplate, err := createCertTemplate("etcd-ca")
	if err != nil {
		return nil, err
	}
	caTemplate.IsCA = true
	caTemplate.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	caTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}

	return caTemplate, nil
}

func createCertTemplate(commonName string) (*x509.Certificate, error) {
	serialNumber, err := generateSerialNumber()
	if err != nil {
		return nil, err
	}
	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:            []string{"DE"},
			Organization:       []string{"SAP SE"},
			OrganizationalUnit: []string{"Gardener"},
			Locality:           []string{"Walldorf"},
			StreetAddress:      []string{"BW"},
			CommonName:         commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * time.Minute),
		BasicConstraintsValid: true,
	}, nil
}

func generateSerialNumber() (*big.Int, error) {
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, max)
}
