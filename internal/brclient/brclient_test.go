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

package brclient

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gardener/etcd-wrapper/internal/types"

	. "github.com/onsi/gomega"
)

var (
	testdataPath       = "../testdata"
	etcdCACertFilePath = filepath.Join(testdataPath, "ca.pem")
)

func TestSuite(t *testing.T) {
	allTests := []struct {
		name   string
		testFn func(t *testing.T, etcdConfigFilePath string)
	}{
		{"getEtcdConfig", testGetEtcdConfig},
		{"getInitializationStatus", testGetInitializationStatus},
		{"triggerInitializer", testTriggerInitialization},
		{"createClient", testCreateSidecarClient},
		{"createTLSConfig", testCreateTLSConfig},
	}

	for _, entry := range allTests {
		t.Run(entry.name, func(t *testing.T) {
			etcdConfigFilePath := createEtcdConfigTempFile(t)
			defer deleteEtcdConfigTempFile(t, etcdConfigFilePath)
			entry.testFn(t, etcdConfigFilePath)
		})
	}
}

func createEtcdConfigTempFile(t *testing.T) string {
	g := NewWithT(t)
	etcdConfigFile, err := os.CreateTemp("", "etcd-config.*.yaml")
	g.Expect(err).To(BeNil())
	return etcdConfigFile.Name()
}

func deleteEtcdConfigTempFile(t *testing.T, etcdConfigFilePath string) {
	g := NewWithT(t)
	if _, err := os.Stat(etcdConfigFilePath); err == nil {
		err = os.Remove(etcdConfigFilePath)
		g.Expect(err).To(BeNil())
	}
}

type TestRoundTripper func(req *http.Request) *http.Response

func (f TestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func testGetEtcdConfig(t *testing.T, etcdConfigFilePath string) {
	table := []struct {
		description             string
		responseCode            int
		responseBody            []byte
		validSidecarBaseAddress bool
		expectError             bool
	}{
		{"test: 200 response code should return a valid etcd config", http.StatusOK, []byte("give me a valid etcd config"), true, false},
		{"test: 202 response code should return a valid etcd config", http.StatusAccepted, []byte("give me a valid etcd config"), true, false},
		{"test: 201 response code should return a valid etcd config", http.StatusCreated, []byte("give me a valid etcd config"), true, false},
		{"test: 208 response code should return an error", http.StatusAlreadyReported, []byte("give me a valid etcd config"), true, true},
		{"test: 400 response code should return an error", http.StatusBadRequest, []byte("give me a valid etcd config"), true, true},
		{"test: should return an error when sidecar base address is invalid", http.StatusBadRequest, []byte("invalid server response"), false, true},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Log(entry.description)
		var sidecarBaseAddress string
		if entry.validSidecarBaseAddress {
			sidecarBaseAddress = ""
		} else {
			sidecarBaseAddress = "//~*wrong{}"
		}

		httpClient := getTestHttpClient(entry.responseCode, entry.responseBody)
		brc := NewClient(httpClient, sidecarBaseAddress, etcdConfigFilePath)
		req, err := brc.GetEtcdConfig(context.TODO())
		if entry.expectError {
			g.Expect(err).ToNot(BeNil())
			g.Expect(req).To(Equal(""))
		} else {
			g.Expect(err).To(BeNil())
			g.Expect(req).To(Equal(etcdConfigFilePath))
		}
	}
}

func testGetInitializationStatus(t *testing.T, etcdConfigFilePath string) {
	table := []struct {
		description             string
		responseCode            int
		responseBody            []byte
		validSidecarBaseAddress bool
		expectError             bool
		expectedStatus          InitStatus
	}{
		{"test: `New` initialization status returned by server should result in `New`", http.StatusOK, []byte(New.String()), true, false, New},
		{"test: `InProgress` initialization status returned by server should result in `InProgress`", http.StatusOK, []byte(InProgress.String()), true, false, InProgress},
		{"test: `Successful` initialization status returned by server should result in  Successful", http.StatusOK, []byte(Successful.String()), true, false, Successful},
		{"test: Unknown initialization status returned by server should result in `InProgress`", http.StatusOK, []byte("error response"), true, false, InProgress},
		{"test: bad response from server should result in `Unknown`", http.StatusBadRequest, []byte("error response"), true, true, Unknown},
		{"test: when sidecar base address is invalid should return an error and result in `Unknown`", http.StatusBadRequest, []byte("error response"), false, true, Unknown},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Log(entry.description)
		var sidecarBaseAddress string
		if entry.validSidecarBaseAddress {
			sidecarBaseAddress = ""
		} else {
			sidecarBaseAddress = "//~*wrong{}"
		}

		httpClient := getTestHttpClient(entry.responseCode, entry.responseBody)
		brc := NewClient(httpClient, sidecarBaseAddress, etcdConfigFilePath)
		req, err := brc.GetInitializationStatus(context.TODO())
		g.Expect(err != nil).To(Equal(entry.expectError))
		g.Expect(req).To(Equal(entry.expectedStatus))
	}
}

func testTriggerInitialization(t *testing.T, etcdConfigFilePath string) {
	table := []struct {
		description             string
		responseCode            int
		responseBody            []byte
		validSidecarBaseAddress bool
		expectError             bool
	}{
		{"test: server returning a valid response should not result in an error", http.StatusOK, []byte("valid server response"), true, false},
		{"test: server returning an error code should result in an error", http.StatusBadRequest, []byte("invalid server response"), true, true},
		{"test: should return an error when sidecar base address is invalid", http.StatusBadRequest, []byte("invalid server response"), false, true},
	}

	for _, entry := range table {
		t.Log(entry.description)
		g := NewWithT(t)
		var sidecarBaseAddress string
		if entry.validSidecarBaseAddress {
			sidecarBaseAddress = ""
		} else {
			sidecarBaseAddress = "//~*wrong{}"
		}

		httpClient := getTestHttpClient(entry.responseCode, entry.responseBody)
		brc := NewClient(httpClient, sidecarBaseAddress, etcdConfigFilePath)
		err := brc.TriggerInitialization(context.TODO(), FullValidation)
		g.Expect(err != nil).To(Equal(entry.expectError))
	}
}

func testCreateSidecarClient(t *testing.T, _ string) {
	incorrectCAFilePath := testdataPath + "/wrong-path"
	table := []struct {
		description   string
		sidecarConfig types.BackupRestoreConfig
		expectError   bool
	}{
		{"test: return error when incorrect sidecar config (CA filepath) is passed", types.BackupRestoreConfig{TLSEnabled: true, CaCertBundlePath: &incorrectCAFilePath}, true},
		{"test: return etcd client when valid sidecar config is passed", types.BackupRestoreConfig{TLSEnabled: true, CaCertBundlePath: &etcdCACertFilePath}, false},
	}
	g := NewWithT(t)
	for _, entry := range table {
		t.Log(entry.description)
		_, err := createClient(entry.sidecarConfig)
		g.Expect(err != nil).To(Equal(entry.expectError))
	}
}

func testCreateTLSConfig(t *testing.T, _ string) {
	incorrectCAFilePath := testdataPath + "/wrong-path"
	table := []struct {
		description   string
		sidecarConfig types.BackupRestoreConfig
		expectError   bool
	}{
		{"test: return valid insecure TLS config when sidecar config does not have TLS enabled", types.BackupRestoreConfig{TLSEnabled: false}, false},
		{"test: return valid TLS config when sidecar config has TLS enabled and valid CA file path", types.BackupRestoreConfig{TLSEnabled: true, CaCertBundlePath: &etcdCACertFilePath}, false},
		{"test: return error when sidecar config has TLS enabled and invalid CA file path", types.BackupRestoreConfig{TLSEnabled: true, CaCertBundlePath: &incorrectCAFilePath}, true},
	}

	g := NewWithT(t)
	for _, entry := range table {
		t.Log(entry.description)
		_, err := createTLSConfig(entry.sidecarConfig)
		g.Expect(err != nil).To(Equal(entry.expectError))
	}
}

func TestNewDefaultClient(t *testing.T) {
	incorrectCAFilePath := etcdCACertFilePath + "/wrong-path"
	table := []struct {
		description   string
		sidecarConfig types.BackupRestoreConfig
		expectError   bool
	}{
		{"test: return error when incorrect sidecar config is passed", types.BackupRestoreConfig{TLSEnabled: true, CaCertBundlePath: &incorrectCAFilePath}, true},
		{"test: return backuprestore client when valid sidecar config is passed", types.BackupRestoreConfig{TLSEnabled: true, CaCertBundlePath: &etcdCACertFilePath}, false},
	}
	g := NewWithT(t)
	for _, entry := range table {
		t.Log(entry.description)
		_, err := NewDefaultClient(entry.sidecarConfig, DefaultEtcdConfigFilePath)
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
