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
