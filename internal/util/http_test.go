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
