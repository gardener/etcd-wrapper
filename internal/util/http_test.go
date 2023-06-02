package util

import (
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
)

func TestResponseHasOKCode(t *testing.T) {
	table := []struct {
		description  string
		responseCode int
		expectValue  bool
	}{
		{"testutil: 200 response code should return true", http.StatusOK, true},
		{"testutil: 201 response code should return true", http.StatusCreated, true},
		{"testutil: 202 response code should return true", http.StatusAccepted, true},
		{"testutil: 400 response code should return true", http.StatusBadRequest, false},
	}

	for _, entry := range table {
		t.Log(entry.description)
		g := NewWithT(t)

		okCode := ResponseHasOKCode(&http.Response{StatusCode: entry.responseCode})
		g.Expect(okCode).To(Equal(entry.expectValue))
	}
}

func TestConstructBaseAddress(t *testing.T) {
	table := []struct {
		description         string
		tlsEnabled          bool
		hostPort            string
		expectedBaseAddress string
	}{
		{"testutil: tls is enabled", true, "localhost:8080", "https://localhost:8080"},
		{"testutil: tls is disabled", false, ":2379", "http://:2379"},
	}

	for _, entry := range table {
		t.Log(entry.description)
		g := NewWithT(t)

		baseAddress := ConstructBaseAddress(entry.tlsEnabled, entry.hostPort)
		g.Expect(baseAddress).To(Equal(entry.expectedBaseAddress))
	}
}
