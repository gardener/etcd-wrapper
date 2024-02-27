// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gardener/etcd-wrapper/internal/testutil"
	. "github.com/onsi/gomega"
)

var (
	testdataPath       = "testdata"
	etcdCACertFilePath = filepath.Join(testdataPath, "ca.pem")
	etcdClientCertPath = filepath.Join(testdataPath, "etcd-01-client.pem")
	etcdClientKeyPath  = filepath.Join(testdataPath, "etcd-01-client-key.pem")
)

func TestCreateCACertPool(t *testing.T) {
	table := []struct {
		description       string
		trustedCAFilePath string
		expectError       bool
	}{
		{"should return error when empty ca cert file path is passed", "", true},
		{"should return error when wrong ca cert file path is passed", testdataPath + "/wrong-path", true},
		{"should not return error when valid ca cert file path is passed", etcdCACertFilePath, false},
	}
	g := NewWithT(t)
	defer func() {
		g.Expect(os.RemoveAll(testdataPath)).To(BeNil())
	}()

	createTLSResources(g)
	for _, entry := range table {
		t.Log(entry.description)
		_, err := CreateCACertPool(entry.trustedCAFilePath)
		g.Expect(err != nil).To(Equal(entry.expectError))
	}
}

func TestCreateTLSConfigWhenTLSDisabled(t *testing.T) {
	g := NewWithT(t)
	tlsConfig, err := CreateTLSConfig(alwaysReturnsFalse, "", "", nil)
	g.Expect(err).To(BeNil())
	g.Expect(tlsConfig.InsecureSkipVerify).To(BeTrue())
}

func TestCreateTLSConfig(t *testing.T) {
	g := NewWithT(t)
	table := []struct {
		description string
		serverName  string
		caCertPath  string
		keyPair     *KeyPair
		expectError bool
	}{
		{"should error out due to wrong CA cert path", "etcd-main-local", etcdCACertFilePath + "/wrong-path", nil, true},
		{"should successfully create valid TLS config with only CA cert", "etcd-main-local", etcdCACertFilePath, nil, false},
		{"should successfully create valid TLS config with CA cert and client cert-key pair", "etcd-main-local", etcdCACertFilePath, &KeyPair{CertPath: etcdClientCertPath, KeyPath: etcdClientKeyPath}, false},
		{"should error out due to wrong cert path", "etcd-main-local", etcdCACertFilePath, &KeyPair{CertPath: etcdClientCertPath + "/wrong-path", KeyPath: etcdClientKeyPath}, true},
	}

	defer func() {
		g.Expect(os.RemoveAll(testdataPath)).To(BeNil())
	}()
	createTLSResources(g)

	for _, entry := range table {
		tlsConfig, err := CreateTLSConfig(alwaysReturnsTrue, entry.serverName, entry.caCertPath, entry.keyPair)
		g.Expect(err != nil).To(Equal(entry.expectError))
		if err == nil {
			g.Expect(tlsConfig.ServerName).To(Equal(entry.serverName))
		}
	}
}

func alwaysReturnsTrue() bool {
	return true
}

func alwaysReturnsFalse() bool {
	return false
}

func createTLSResources(g *WithT) {
	var (
		err                              error
		caCertKeyPair, clientCertKeyPair *testutil.CertKeyPair
		tlsResCreator                    *testutil.TLSResourceCreator
	)
	if _, err = os.Stat(testdataPath); errors.Is(err, os.ErrNotExist) {
		g.Expect(os.Mkdir(testdataPath, os.ModeDir|os.ModePerm)).To(Succeed())
	}
	tlsResCreator, err = testutil.NewTLSResourceCreator()
	g.Expect(err).To(BeNil())
	// create and write CA certificate and private key
	caCertKeyPair, err = tlsResCreator.CreateCACertAndKey()
	g.Expect(err).To(BeNil())
	g.Expect(caCertKeyPair.EncodeAndWrite(testdataPath, "ca.pem", "ca-key.pem")).To(Succeed())

	// create and write client certificate and key
	clientCertKeyPair, err = tlsResCreator.CreateETCDClientCertAndKey()
	g.Expect(err).To(BeNil())
	g.Expect(clientCertKeyPair.EncodeAndWrite(testdataPath, "etcd-01-client.pem", "etcd-01-client-key.pem")).To(Succeed())
}
