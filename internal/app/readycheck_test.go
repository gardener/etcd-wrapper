// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gardener/etcd-wrapper/internal/testutil"
	"github.com/gardener/etcd-wrapper/internal/types"

	"go.etcd.io/etcd/embed"
	"go.uber.org/zap"

	. "github.com/onsi/gomega"
)

var (
	testdataPath       = "testdata"
	etcdCertFilePath   = filepath.Join(testdataPath, "etcd-01.pem")
	etcdCACertFilePath = filepath.Join(testdataPath, "ca.pem")
	etcdKeyFilePath    = filepath.Join(testdataPath, "etcd-01-key.pem")
)

func TestSuit(t *testing.T) {
	allTests := []struct {
		name   string
		testFn func(t *testing.T)
	}{
		{"queryAndUpdateEtcdReadiness", testQueryEtcdReadiness},
		{"readinessHandler", testReadinessHandler},
		{"createEtcdClient", testCreateEtcdClient},
		{"isTLSEnabled", testIsTLSEnabled},
	}

	g := NewWithT(t)
	createTLSResources(g)
	defer func() {
		g.Expect(os.RemoveAll(testdataPath)).To(BeNil())
	}()

	for _, entry := range allTests {
		t.Run(entry.name, entry.testFn)
	}
}

func testQueryEtcdReadiness(t *testing.T) {
	table := []struct {
		description  string
		querySuccess bool
		expectStatus bool
	}{
		{"etcd ready status should be set to true when etcd query succeeds", true, true},
		{"etcd ready status should be set to false when etcd query fails", false, false},
	}

	for _, entry := range table {
		t.Log(entry.description)
		g := NewWithT(t)

		ctx, cancel := context.WithCancel(context.Background())
		app := createApplicationInstance(ctx, cancel, g)

		cli, err := app.createEtcdClient()
		g.Expect(err).To(BeNil())
		if entry.querySuccess {
			fakeKV := EtcdFakeKV{}
			cli.KV = &fakeKV
		}
		app.etcdClient = cli
		g.Expect(app.isEtcdReady()).To(Equal(entry.expectStatus))

		app.Close()
	}
}

func testReadinessHandler(t *testing.T) {
	table := []struct {
		description    string
		readyStatus    bool
		expectedStatus int
	}{
		{"should return http.StatusOK when etcdStatus.Ready is set to true", true, http.StatusOK},
		{"should return http.StatusServiceUnavailable when etcdStatus.Ready is set to false", false, http.StatusServiceUnavailable},
	}

	for _, entry := range table {
		t.Log(entry.description)
		g := NewWithT(t)

		ctx, cancel := context.WithCancel(context.Background())
		app := createApplicationInstance(ctx, cancel, g)
		app.etcdReady = entry.readyStatus

		request, err := http.NewRequest("GET", "/readyz", nil)
		g.Expect(err).To(BeNil())
		response := httptest.NewRecorder()
		handler := http.HandlerFunc(app.readinessHandler)
		handler.ServeHTTP(response, request)
		g.Expect(response.Code).To(Equal(entry.expectedStatus))

		app.Close()
	}
}

func testCreateEtcdClient(t *testing.T) {
	table := []struct {
		description       string
		certFilePath      string
		keyFilePath       string
		trustedCAFilePath string
		expectError       bool
		endpointScheme    string
	}{
		{"should return valid etcd client with HTTP scheme when no certificates are passed", "", "", "", false, "http"},
		{"should return valid etcd client with HTTPS scheme when all certificates are passed", etcdCertFilePath, etcdKeyFilePath, etcdCACertFilePath, false, "https"},
		{"should return valid etcd client with HTTP scheme when empty certificate file path is passed", "", etcdKeyFilePath, etcdCACertFilePath, false, "http"},
		{"should return valid etcd client with HTTP scheme when empty key file path is passed", etcdCertFilePath, "", etcdCACertFilePath, false, "http"},
		{"should return valid etcd client with HTTP scheme when empty CA cert file path is passed", etcdCertFilePath, etcdKeyFilePath, "", false, "http"},
		{"should return error when wrong certificate file path is passed", filepath.Join(testdataPath, "does-not-exist.crt"), etcdKeyFilePath, etcdCACertFilePath, true, ""},
	}

	g := NewWithT(t)
	// create testutil cert files
	for _, entry := range table {
		t.Log(entry.description)

		ctx, cancel := context.WithCancel(context.Background())
		app := createApplicationInstance(ctx, cancel, g)
		app.cfg.ClientTLSInfo.CertFile = entry.certFilePath
		app.cfg.ClientTLSInfo.KeyFile = entry.keyFilePath
		app.cfg.ClientTLSInfo.TrustedCAFile = entry.trustedCAFilePath
		app.Config.EtcdClientTLS.CertPath = entry.certFilePath
		app.Config.EtcdClientTLS.KeyPath = entry.keyFilePath
		app.Config.EtcdClientTLS.ServerName = app.Config.BackupRestore.GetHost()

		etcdClient, err := app.createEtcdClient()
		g.Expect(err != nil).To(Equal(entry.expectError))
		if !entry.expectError {
			g.Expect(strings.Split(etcdClient.Endpoints()[0], "://")[0]).To(Equal(entry.endpointScheme))
		}

		app.Close()
	}
}

func testIsTLSEnabled(t *testing.T) {
	table := []struct {
		description       string
		certFilePath      string
		keyFilePath       string
		trustedCAFilePath string
		expectedResult    bool
	}{
		{"should return true when all files are present", "testutil/path/for/certFile", "testutil/path/for/keyFile", "testutil/path/for/trustedCAFile", true},
		{"should return false when only certFile is not present", "", "testutil/path/for/keyFile", "testutil/path/for/trustedCAFile", false},
		{"should return false when only keyFile is not present", "testutil/path/for/certFile", "", "testutil/path/for/trustedCAFile", false},
		{"should return false when only trusterCAFile is not present", "testutil/path/for/certFile", "testutil/path/for/keyFile", "", false},
		{"should return true when all files not are present", "", "", "", false},
	}

	for _, entry := range table {
		t.Log(entry.description)
		g := NewWithT(t)

		ctx, cancel := context.WithCancel(context.Background())
		app := createApplicationInstance(ctx, cancel, g)
		app.cfg.ClientTLSInfo.CertFile = entry.certFilePath
		app.cfg.ClientTLSInfo.KeyFile = entry.keyFilePath
		app.cfg.ClientTLSInfo.TrustedCAFile = entry.trustedCAFilePath

		g.Expect(app.isTLSEnabled()).To(Equal(entry.expectedResult))

		app.Close()

	}
}

func createApplicationInstance(ctx context.Context, cancelFn context.CancelFunc, g *GomegaWithT) *Application {
	config := types.Config{
		BackupRestore: types.BackupRestoreConfig{
			HostPort:   ":2379",
			TLSEnabled: false,
		},
	}
	app, err := NewApplication(ctx, cancelFn, config, time.Minute, zap.NewExample())
	g.Expect(err).To(BeNil())
	app.cfg = &embed.Config{}
	cli, err := app.createEtcdClient()
	g.Expect(err).To(BeNil())
	app.etcdClient = cli
	return app
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
	g.Expect(clientCertKeyPair.EncodeAndWrite(testdataPath, "etcd-01.pem", "etcd-01-key.pem")).To(Succeed())
}
