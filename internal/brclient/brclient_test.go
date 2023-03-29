package brclient

import (
	"bytes"
	"context"
	"github.com/gardener/etcd-wrapper/internal/types"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

var (
	pwd, _             = os.Getwd()
	etcdConfigFilePath = pwd + "/../../test/etcd-config.yaml"
	caFilePath         = pwd + "/../../test/CAFile.crt"
	// sample ca file taken from https://go.dev/src/crypto/x509/x509_test.go
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

func TestSuit(t *testing.T) {
	allTests := []struct {
		name   string
		testFn func(t *testing.T)
	}{
		{"getEtcdConfig", testGetEtcdConfig},
		{"getInitializationStatus", testGetInitializationStatus},
		{"triggerInitializer", testTriggerInitialization},
		{"createSidecarClient", testCreateSidecarClient},
		{"createTLSConfig", testCreateTLSConfig},
	}

	for _, entry := range allTests {
		t.Run(entry.name, entry.testFn)
	}
}

type TestRoundTripper func(req *http.Request) *http.Response

func (f TestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func testGetEtcdConfig(t *testing.T) {
	table := []struct {
		description  string
		responseCode int
		responseBody []byte
		expectError  bool
	}{
		{"test: 200 response code should return a valid etcd config", http.StatusOK, []byte("give me a valid etcd config"), false},
		{"test: 202 response code should return a valid etcd config", http.StatusAccepted, []byte("give me a valid etcd config"), false},
		{"test: 201 response code should return a valid etcd config", http.StatusCreated, []byte("give me a valid etcd config"), false},
		{"test: 208 response code should return an error", http.StatusAlreadyReported, []byte("give me a valid etcd config"), true},
		{"test: 400 response code should return an error", http.StatusBadRequest, []byte("give me a valid etcd config"), true},
	}

	g := NewWithT(t)
	defer deleteTestFiles(g)
	for _, entry := range table {
		t.Log(entry.description)
		httpClient := getTestHttpClient(entry.responseCode, entry.responseBody)
		brclient, err := NewClient(httpClient, "", etcdConfigFilePath)
		g.Expect(err).To(BeNil())
		req, err := brclient.GetEtcdConfig(context.TODO())
		if entry.expectError {
			g.Expect(err).ToNot(BeNil())
			g.Expect(req).To(Equal(""))
		} else {
			g.Expect(err).To(BeNil())
			g.Expect(req).To(Equal(etcdConfigFilePath))
		}
	}
}

func testGetInitializationStatus(t *testing.T) {
	table := []struct {
		description    string
		responseCode   int
		responseBody   []byte
		expectError    bool
		expectedStatus InitStatus
	}{
		{"test: `New` initialization status returned by server should result in `New`", http.StatusOK, []byte(New.String()), false, New},
		{"test: `InProgress` initialization status returned by server should result in `InProgress`", http.StatusOK, []byte(InProgress.String()), false, InProgress},
		{"test: `Successful` initialization status returned by server should result in  Successful", http.StatusOK, []byte(Successful.String()), false, Successful},
		{"test: Unknown initialization status returned by server should result in `InProgress`", http.StatusOK, []byte("error response"), false, InProgress},
		{"test: bad response from server should result in `Unknown`", http.StatusBadRequest, []byte("error response"), true, Unknown},
	}

	g := NewWithT(t)
	defer deleteTestFiles(g)
	for _, entry := range table {
		t.Log(entry.description)
		httpClient := getTestHttpClient(entry.responseCode, entry.responseBody)
		brclient, err := NewClient(httpClient, "", etcdConfigFilePath)
		g.Expect(err).To(BeNil())
		req, err := brclient.GetInitializationStatus(context.TODO())
		g.Expect(err != nil).To(Equal(entry.expectError))
		g.Expect(req).To(Equal(entry.expectedStatus))
	}
}

func testTriggerInitialization(t *testing.T) {
	table := []struct {
		description  string
		responseCode int
		responseBody []byte
		expectError  bool
	}{
		{"test: server returning a valid response should not result in an error", http.StatusOK, []byte("valid server response"), false},
		{"test: server returning an error code should result in an error", http.StatusBadRequest, []byte("invalid server response"), true},
	}

	for _, entry := range table {
		t.Log(entry.description)
		g := NewWithT(t)
		httpClient := getTestHttpClient(entry.responseCode, entry.responseBody)
		brclient, err := NewClient(httpClient, "", etcdConfigFilePath)
		g.Expect(err).To(BeNil())
		err = brclient.TriggerInitialization(context.TODO(), FullValidation)
		g.Expect(err != nil).To(Equal(entry.expectError))
	}
}

func TestNewDefaultClient(t *testing.T) {
	incorrectCAFilePath := caFilePath + "/wrong-path"
	table := []struct {
		description   string
		sidecarConfig types.SidecarConfig
		expectError   bool
	}{
		{"test: return error when incorrect sidecar config is passed", types.SidecarConfig{TLSEnabled: true, CaCertBundlePath: &incorrectCAFilePath}, true},
		{"test: return backuprestore client when valid sidecar config is passed", types.SidecarConfig{TLSEnabled: true, CaCertBundlePath: &caFilePath}, false},
	}
	g := NewWithT(t)
	createTestCACertFiles(g)
	defer deleteTestFiles(g)
	for _, entry := range table {
		t.Log(entry.description)
		_, err := NewDefaultClient(entry.sidecarConfig, DefaultEtcdConfigFilePath)
		g.Expect(err != nil).To(Equal(entry.expectError))
	}
}

func testCreateSidecarClient(t *testing.T) {
	incorrectCAFilePath := caFilePath + "/wrong-path"
	table := []struct {
		description   string
		sidecarConfig types.SidecarConfig
		expectError   bool
	}{
		{"test: return error when incorrect sidecar config is passed", types.SidecarConfig{TLSEnabled: true, CaCertBundlePath: &incorrectCAFilePath}, true},
		{"test: return etcd client when valid sidecar config is passed", types.SidecarConfig{TLSEnabled: true, CaCertBundlePath: &caFilePath}, false},
	}
	g := NewWithT(t)
	createTestCACertFiles(g)
	defer deleteTestFiles(g)
	for _, entry := range table {
		t.Log(entry.description)
		_, err := createSidecarClient(entry.sidecarConfig)
		g.Expect(err != nil).To(Equal(entry.expectError))
	}
}

func testCreateTLSConfig(t *testing.T) {
	incorrectCAFilePath := caFilePath + "/wrong-path"
	table := []struct {
		description   string
		sidecarConfig types.SidecarConfig
		expectError   bool
	}{
		{"test: return valid insecure TLS config when sidecar config does not have TLS enabled", types.SidecarConfig{TLSEnabled: false}, false},
		{"test: return valid TLS config when sidecar config has TLS enabled and valid CA file path", types.SidecarConfig{TLSEnabled: true, CaCertBundlePath: &caFilePath}, false},
		{"test: return error when sidecar config has TLS enabled and invalid CA file path", types.SidecarConfig{TLSEnabled: true, CaCertBundlePath: &incorrectCAFilePath}, true},
	}

	g := NewWithT(t)
	createTestCACertFiles(g)
	defer deleteTestFiles(g)
	for _, entry := range table {
		t.Log(entry.description)
		_, err := createTLSConfig(entry.sidecarConfig)
		g.Expect(err != nil).To(Equal(entry.expectError))
	}
}

func getTestHttpClient(responseCode int, responseBody []byte) *http.Client {
	return &http.Client{
		Transport: TestRoundTripper(func(req *http.Request) *http.Response {
			var contentLen int64
			if responseBody != nil {
				contentLen = int64(len(responseBody))
			}
			return &http.Response{
				StatusCode:    responseCode,
				Body:          io.NopCloser(bytes.NewReader(responseBody)),
				ContentLength: contentLen,
			}
		}),
		Timeout: 5 * time.Second,
	}
}

func createTestCACertFiles(g *GomegaWithT) {
	err := os.WriteFile(caFilePath, []byte(ca), 0644)
	g.Expect(err).To(BeNil())
}

func deleteTestFiles(g *GomegaWithT) {
	if _, err := os.Stat(etcdConfigFilePath); err == nil {
		err = os.Remove(etcdConfigFilePath)
		g.Expect(err).To(BeNil())
	}
	if _, err := os.Stat(caFilePath); err == nil {
		err = os.Remove(caFilePath)
		g.Expect(err).To(BeNil())
	}
}
