package app

import (
	"context"
	"github.com/gardener/etcd-wrapper/internal/types"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

var (
	pwd, _       = os.Getwd()
	certFilePath = pwd + "/../../test/certFile.crt"
	keyFilePath  = pwd + "/../../test/keyFile.key"
	caFilePath   = pwd + "/../../test/CAFile.crt"
	// sample cert file taken from https://go.dev/src/crypto/tls/tls_test.go
	cert = `-----BEGIN CERTIFICATE-----
MIIB0zCCAX2gAwIBAgIJAI/M7BYjwB+uMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTIwOTEyMjE1MjAyWhcNMTUwOTEyMjE1MjAyWjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBANLJ
hPHhITqQbPklG3ibCVxwGMRfp/v4XqhfdQHdcVfHap6NQ5Wok/4xIA+ui35/MmNa
rtNuC+BdZ1tMuVCPFZcCAwEAAaNQME4wHQYDVR0OBBYEFJvKs8RfJaXTH08W+SGv
zQyKn0H8MB8GA1UdIwQYMBaAFJvKs8RfJaXTH08W+SGvzQyKn0H8MAwGA1UdEwQF
MAMBAf8wDQYJKoZIhvcNAQEFBQADQQBJlffJHybjDGxRMqaRmDhX0+6v02TUKZsW
r5QuVbpQhH6u+0UgcW0jp9QwpxoPTLTWGXEWBBBurxFwiCBhkQ+V
-----END CERTIFICATE-----
` // sample key file taken from https://go.dev/src/crypto/tls/tls_test.go
	key = `-----BEGIN RSA PRIVATE KEY-----
MIIBOwIBAAJBANLJhPHhITqQbPklG3ibCVxwGMRfp/v4XqhfdQHdcVfHap6NQ5Wo
k/4xIA+ui35/MmNartNuC+BdZ1tMuVCPFZcCAwEAAQJAEJ2N+zsR0Xn8/Q6twa4G
6OB1M1WO+k+ztnX/1SvNeWu8D6GImtupLTYgjZcHufykj09jiHmjHx8u8ZZB/o1N
MQIhAPW+eyZo7ay3lMz1V01WVjNKK9QSn1MJlb06h/LuYv9FAiEA25WPedKgVyCW
SmUwbPw8fnTcpqDWE3yTO3vKcebqMSsCIBF3UmVue8YU3jybC3NxuXq3wNm34R8T
xVLHwDXh/6NJAiEAl2oHGGLz64BuAfjKrqwz7qMYr9HCLIe/YsoWq/olzScCIQDi
D2lWusoe2/nEqfDVVWGWlyJ7yOmqaVm/iNUN9B2N2g==
-----END RSA PRIVATE KEY-----
` // sample ca file taken from https://go.dev/src/crypto/x509/x509_test.go
	ca = `
-----BEGIN CERTIFICATE-----
MIIDBjCCAe6gAwIBAgIRANXM5I3gjuqDfTp/PYrs+u8wDQYJKoZIhvcNAQELBQAw
EjEQMA4GA1UEChMHQWNtZSBDbzAeFw0xODAzMjcxOTU2MjFaFw0xOTAzMjcxOTU2
MjFaMBIxEDAOBgNVBAoTB0FjbWUgQ28wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQDK+9m3rjsO2Djes6bIYQZ3eV29JF09ZrjOrEHLtaKrD6/acsoSoTsf
cQr+rzzztdB5ijWXCS64zo/0OiqBeZUNZ67jVdToa9qW5UYe2H0Y+ZNdfA5GYMFD
yk/l3/uBu3suTZPfXiW2TjEi27Q8ruNUIZ54DpTcs6y2rBRFzadPWwn/VQMlvRXM
jrzl8Y08dgnYmaAHprxVzwMXcQ/Brol+v9GvjaH1DooHqkn8O178wsPQNhdtvN01
IXL46cYdcUwWrE/GX5u+9DaSi+0KWxAPQ+NVD5qUI0CKl4714yGGh7feXMjJdHgl
VG4QJZlJvC4FsURgCHJT6uHGIelnSwhbAgMBAAGjVzBVMA4GA1UdDwEB/wQEAwIF
oDATBgNVHSUEDDAKBggrBgEFBQcDATAMBgNVHRMBAf8EAjAAMCAGA1UdEQQZMBeC
FVRlc3RTeXN0ZW1DZXJ0UG9vbC5nbzANBgkqhkiG9w0BAQsFAAOCAQEAwuSRx/VR
BKh2ICxZjL6jBwk/7UlU1XKbhQD96RqkidDNGEc6eLZ90Z5XXTurEsXqdm5jQYPs
1cdcSW+fOSMl7MfW9e5tM66FaIPZl9rKZ1r7GkOfgn93xdLAWe8XHd19xRfDreub
YC8DVqgLASOEYFupVSl76ktPfxkU5KCvmUf3P2PrRybk1qLGFytGxfyice2gHSNI
gify3K/+H/7wCkyFW4xYvzl7WW4mXxoqPRPjQt1J423DhnnQ4G1P8V/vhUpXNXOq
N9IEPnWuihC09cyx/WMQIUlWnaQLHdfpPS04Iez3yy2PdfXJzwfPrja7rNE+skK6
pa/O1nF0AfWOpw==
-----END CERTIFICATE-----
	`
)

// EtcdFakeKV mocks the KV interface of etcd
// required to mock etcd get calls
type EtcdFakeKV struct{}

func (c *EtcdFakeKV) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	return nil, nil
}
func (c *EtcdFakeKV) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	return nil, nil
}
func (c *EtcdFakeKV) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	return nil, nil
}
func (c *EtcdFakeKV) Compact(ctx context.Context, rev int64, opts ...clientv3.CompactOption) (*clientv3.CompactResponse, error) {
	return nil, nil
}
func (c *EtcdFakeKV) Txn(ctx context.Context) clientv3.Txn {
	return nil
}
func (c *EtcdFakeKV) Do(ctx context.Context, op clientv3.Op) (clientv3.OpResponse, error) {
	return clientv3.OpResponse{}, nil
}

func TestSuit(t *testing.T) {
	allTests := []struct {
		name   string
		testFn func(t *testing.T)
	}{
		{"queryEtcdReadiness", testQueryEtcdReadiness},
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

		etcdStatus.ready = false
		go app.queryEtcdReadiness()
		<-time.After(10 * time.Second)
		g.Expect(etcdStatus.ready).To(Equal(entry.expectStatus))

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
		etcdStatus.ready = entry.readyStatus
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
		{"test: should return valid etcd client with HTTPS scheme when all certificates are passed", certFilePath, keyFilePath, caFilePath, false, schemeHTTPS},
		{"test: should return valid etcd client with HTTP scheme when empty certificate file path is passed", "", keyFilePath, caFilePath, false, schemeHTTP},
		{"test: should return valid etcd client with HTTP scheme when empty key file path is passed", caFilePath, "", caFilePath, false, schemeHTTP},
		{"test: should return valid etcd client with HTTP scheme when empty CA cert file path is passed", "", keyFilePath, "", false, schemeHTTP},
		{"test: should return error when wrong certificate file path is passed", certFilePath + "/wrong-path", keyFilePath, caFilePath, true, ""},
	}

	g := NewWithT(t)
	// create test cert files
	createTestCertFiles(g)
	defer deleteTestCertFiles(g)
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
		{"test: should return valid HTTP TLS config when certificate file path is not passed", "", keyFilePath, caFilePath, false, true},
		{"test: should return valid HTTP TLS config when key file path is not passed", certFilePath, "", caFilePath, false, true},
		{"test: should return valid HTTP TLS config when CA file path is not passed", certFilePath, keyFilePath, "", false, true},
		{"test: should return valid HTTPS TLS config when all certificates are passes", certFilePath, keyFilePath, caFilePath, false, false},
		{"test: should return error when incorrect certificate file path is passed", certFilePath + "/wrong-path", keyFilePath, caFilePath, true, true},
		{"test: should return error when incorrect key file path is passed", certFilePath, keyFilePath + "/wrong-path", caFilePath, true, true},
		{"test: should return error when incorrect CA file path is passed", certFilePath, keyFilePath, caFilePath + "/wrong-path", true, true},
	}

	g := NewWithT(t)
	// create test cert files
	createTestCertFiles(g)
	defer deleteTestCertFiles(g)

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

func createTestCertFiles(g *GomegaWithT) {
	err := os.WriteFile(certFilePath, []byte(cert), 0644)
	g.Expect(err).To(BeNil())
	err = os.WriteFile(keyFilePath, []byte(key), 0644)
	g.Expect(err).To(BeNil())
	err = os.WriteFile(caFilePath, []byte(ca), 0644)
	g.Expect(err).To(BeNil())
}

func deleteTestCertFiles(g *GomegaWithT) {
	err := os.Remove(certFilePath)
	g.Expect(err).To(BeNil())
	err = os.Remove(keyFilePath)
	g.Expect(err).To(BeNil())
	err = os.Remove(caFilePath)
	g.Expect(err).To(BeNil())
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
