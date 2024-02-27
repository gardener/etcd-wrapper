// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
		{"200 response code should return true", http.StatusOK, true},
		{"201 response code should return true", http.StatusCreated, true},
		{"202 response code should return true", http.StatusAccepted, true},
		{"400 response code should return true", http.StatusBadRequest, false},
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
		{"tls is enabled", true, "localhost:8080", "https://localhost:8080"},
		{"tls is disabled", false, ":2379", "http://:2379"},
	}

	for _, entry := range table {
		t.Log(entry.description)
		g := NewWithT(t)

		baseAddress := ConstructBaseAddress(entry.tlsEnabled, entry.hostPort)
		g.Expect(baseAddress).To(Equal(entry.expectedBaseAddress))
	}
}
