// Copyright 2023 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gardener/etcd-wrapper/internal/types"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"

	. "github.com/onsi/gomega"
)

var (
	testdataPath       = "../testdata"
	etcdCertFilePath   = filepath.Join(testdataPath, "etcd.crt")
	etcdCACertFilePath = filepath.Join(testdataPath, "ca.crt")
	etcdKeyFilePath    = filepath.Join(testdataPath, "etcd.key")
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
		{"getTLsConfig", testGetTLsConfig},
	}

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
		{"test: etcd ready status should be set to true when etcd query succeeds", true, true},
		{"test: etcd ready status should be set to false when etcd query fails", false, false},
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
		{"test: should return http.StatusOK when etcdStatus.Ready is set to true", true, http.StatusOK},
		{"test: should return http.StatusServiceUnavailable when etcdStatus.Ready is set to false", false, http.StatusServiceUnavailable},
	}

	for _, entry := range table {
		t.Log(entry.description)
		g := NewWithT(t)

		ctx, cancel := context.WithCancel(context.Background())
		etcdReady = entry.readyStatus
		app := createApplicationInstance(ctx, cancel, g)

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
		{"test: should return valid etcd client with HTTP scheme when no certificates are passed", "", "", "", false, schemeHTTP},
		{"test: should return valid etcd client with HTTPS scheme when all certificates are passed", etcdCertFilePath, etcdKeyFilePath, etcdCACertFilePath, false, schemeHTTPS},
		{"test: should return valid etcd client with HTTP scheme when empty certificate file path is passed", "", etcdKeyFilePath, etcdCACertFilePath, false, schemeHTTP},
		{"test: should return valid etcd client with HTTP scheme when empty key file path is passed", etcdCertFilePath, "", etcdCACertFilePath, false, schemeHTTP},
		{"test: should return valid etcd client with HTTP scheme when empty CA cert file path is passed", etcdCertFilePath, etcdKeyFilePath, "", false, schemeHTTP},
		{"test: should return error when wrong certificate file path is passed", filepath.Join(testdataPath, "does-not-exist.crt"), etcdKeyFilePath, etcdCACertFilePath, true, ""},
	}

	g := NewWithT(t)
	// create test cert files
	for _, entry := range table {
		t.Log(entry.description)

		ctx, cancel := context.WithCancel(context.Background())
		app := createApplicationInstance(ctx, cancel, g)
		app.cfg.ClientTLSInfo.CertFile = entry.certFilePath
		app.cfg.ClientTLSInfo.KeyFile = entry.keyFilePath
		app.cfg.ClientTLSInfo.TrustedCAFile = entry.trustedCAFilePath

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
		{"test: should return true when all files are present", "test/path/for/certFile", "test/path/for/keyFile", "test/path/for/trustedCAFile", true},
		{"test: should return false when only certFile is not present", "", "test/path/for/keyFile", "test/path/for/trustedCAFile", false},
		{"test: should return false when only keyFile is not present", "test/path/for/certFile", "", "test/path/for/trustedCAFile", false},
		{"test: should return false when only trusterCAFile is not present", "test/path/for/certFile", "test/path/for/keyFile", "", false},
		{"test: should return true when all files not are present", "", "", "", false},
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

func testGetTLsConfig(t *testing.T) {
	table := []struct {
		description       string
		certFilePath      string
		keyFilePath       string
		trustedCAFilePath string
		expectError       bool
		emptyTLSConfig    bool
	}{
		{"test: should return valid HTTP TLS config when no certificates are passes", "", "", "", false, true},
		{"test: should return valid HTTP TLS config when certificate file path is not passed", "", etcdKeyFilePath, etcdCACertFilePath, false, true},
		{"test: should return valid HTTP TLS config when key file path is not passed", etcdCertFilePath, "", etcdCACertFilePath, false, true},
		{"test: should return valid HTTP TLS config when CA file path is not passed", etcdCertFilePath, etcdKeyFilePath, "", false, true},
		{"test: should return valid HTTPS TLS config when all certificates are passes", etcdCertFilePath, etcdKeyFilePath, etcdCACertFilePath, false, false},
		{"test: should return error when incorrect certificate file path is passed", filepath.Join(testdataPath, "does-not-exist-cert.crt"), etcdKeyFilePath, etcdCACertFilePath, true, true},
		{"test: should return error when incorrect key file path is passed", etcdCertFilePath, filepath.Join(testdataPath, "does-not-exist-key.key"), etcdCACertFilePath, true, true},
		{"test: should return error when incorrect CA file path is passed", etcdCertFilePath, etcdKeyFilePath, filepath.Join(testdataPath, "does-not-exist-ca.crt"), true, true},
	}

	g := NewWithT(t)
	// create test cert files
	for _, entry := range table {
		t.Log(entry.description)

		ctx, cancel := context.WithCancel(context.Background())
		app := createApplicationInstance(ctx, cancel, g)
		app.cfg.ClientTLSInfo.CertFile = entry.certFilePath
		app.cfg.ClientTLSInfo.KeyFile = entry.keyFilePath
		app.cfg.ClientTLSInfo.TrustedCAFile = entry.trustedCAFilePath

		tlsConfig, err := app.getTLSConfig()

		if entry.expectError {
			g.Expect(err).ToNot(BeNil())
			g.Expect(tlsConfig).To(BeNil())
		} else {
			g.Expect(err).To(BeNil())
			g.Expect(tlsConfig.RootCAs == nil).To(Equal(entry.emptyTLSConfig))
		}

		app.Close()

	}
}

func createApplicationInstance(ctx context.Context, cancelFn context.CancelFunc, g *GomegaWithT) *Application {
	sidecarConfig := &types.SidecarConfig{
		HostPort:   ":2379",
		TLSEnabled: false,
	}
	app, err := NewApplication(ctx, cancelFn, sidecarConfig, time.Minute, zap.NewExample())
	g.Expect(err).To(BeNil())
	app.cfg = &embed.Config{
		ClientTLSInfo: transport.TLSInfo{
			CertFile:      "",
			KeyFile:       "",
			TrustedCAFile: "",
		},
	}
	cli, err := app.createEtcdClient()
	g.Expect(err).To(BeNil())
	app.etcdClient = cli
	return app
}
