package util

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

var retry int

var (
	pwd, _     = os.Getwd()
	caFilePath = pwd + "/../../test/CAFile.crt"
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

func TestUtilSuite(t *testing.T) {
	allTests := []struct {
		name   string
		testFn func(t *testing.T)
	}{
		{"ResponseHasOKCode", testResponseHasOKCode},
		{"retry", testRetry},
		{"CreateCACertPool", testCreateCACertPool},
	}

	for _, entry := range allTests {
		t.Run(entry.name, entry.testFn)
	}
}

func testResponseHasOKCode(t *testing.T) {
	table := []struct {
		description  string
		responseCode int
		expectValue  bool
	}{
		{"test: 200 response code should return true", http.StatusOK, true},
		{"test: 201 response code should return true", http.StatusCreated, true},
		{"test: 202 response code should return true", http.StatusAccepted, true},
		{"test: 400 response code should return true", http.StatusBadRequest, false},
	}

	for _, entry := range table {
		t.Log(entry.description)
		g := NewWithT(t)

		okCode := ResponseHasOKCode(&http.Response{StatusCode: entry.responseCode})
		g.Expect(okCode).To(Equal(entry.expectValue))
	}
}

func testRetry(t *testing.T) {
	table := []struct {
		description string
		retryableFn func() (int, error)
		retry       int
		numAttempts int
		canRetryFn  CanRetryPredicate
		expectError bool
	}{
		{"test: should return error if retryable always returns an error and attempts are expired", retryIsZero, 6, 5, alwaysRetry, true},
		{"test: should return error if retryable always returns an error and attempts are not expired", retryIsZero, 6, 5, neverRetry, true},
		{"test: should not return error if retryable returns valid value before attempts expire", retryIsZero, 3, 5, alwaysRetry, false},
	}

	for _, entry := range table {
		t.Log(entry.description)
		g := NewWithT(t)

		logger, _ := zap.NewDevelopment()
		retry = entry.retry
		result := Retry(context.TODO(), logger, entry.description, entry.retryableFn, entry.numAttempts, time.Second, entry.canRetryFn)

		g.Expect(result.Err != nil).To(Equal(entry.expectError))
	}
}

func testCreateCACertPool(t *testing.T) {
	table := []struct {
		description       string
		trustedCAFilePath string
		expectError       bool
	}{
		{"test: should return error when empty ca cert file path is passed", "", true},
		{"test: should return error when wrong ca cert file path is passed", caFilePath + "/wrong-path", true},
		{"test: should not return error when valid ca cert file path is passed", caFilePath, false},
	}

	g := NewWithT(t)
	// create test cert files
	createTestCACertFiles(g)
	defer deleteTestCACertFiles(g)
	for _, entry := range table {
		t.Log(entry.description)
		_, err := CreateCACertPool(entry.trustedCAFilePath)
		g.Expect(err != nil).To(Equal(entry.expectError))
	}
}

func alwaysRetry(_ error) bool {
	return true
}

func neverRetry(_ error) bool {
	return false
}

func retryIsZero() (int, error) {
	retry = retry - 1
	if retry == 0 {
		return retry, nil
	}
	return retry, fmt.Errorf("globalInt is not 0")
}

func createTestCACertFiles(g *GomegaWithT) {
	err := os.WriteFile(caFilePath, []byte(ca), 0644)
	g.Expect(err).To(BeNil())
}

func deleteTestCACertFiles(g *GomegaWithT) {
	err := os.Remove(caFilePath)
	g.Expect(err).To(BeNil())
}
