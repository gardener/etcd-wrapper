package brclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var (
	testClient *http.Client
	testServer *httptest.Server
	httpClient BackupRestoreClient
)

const (
	//TODO @aaronfern: Update this local path
	etcdConfigFilePath = "/Users/I544000/go/src/github.com/gardener/etcd-wrapper/test/etcd-config.yaml"
)

func TestBrClient_GetEtcdConfig(t *testing.T) {
	_, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting present working directory: %v", err)
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "Sample etcd config")
	}
	testServer = httptest.NewServer(http.HandlerFunc(handler))
	testClient = testServer.Client()
	httpClient, err = NewTestClient(testClient, testServer.URL, etcdConfigFilePath)
	if err != nil {
		t.Errorf("Error creating test client: %v", err)
	}
	req, err := httpClient.GetEtcdConfig()
	if err != nil {
		t.Errorf("Error fetching etcd config: %v", err)
	}
	if req == "" {
		t.Error("Invalid config file path returned")
	}
	testServer.Close()
	err = os.Remove(etcdConfigFilePath)
	if err != nil {
		t.Errorf("Error removing etcd config file: %v", err)
	}
}

func TestBrClient_GetInitializationStatus(t *testing.T) {
	tests := []struct {
		name                 string
		description          string
		serverReturnedStatus InitStatus
		expectedStatus       InitStatus
		run                  func(t *testing.T, serverReturnedStatus InitStatus, expectedStatus InitStatus)
	}{
		{"New", "Initialization status returned by server is New", New, New, testGetInitializationStatus},
		{"InProgress", "Initialization status returned by server is InProgress", InProgress, InProgress, testGetInitializationStatus},
		{"Successful", "Initialization status returned by server is Successful", Successful, Successful, testGetInitializationStatus},
		{"Successful", "Initialization status returned by server is who knows", 3, InProgress, testGetInitializationStatus},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			test.run(t, test.serverReturnedStatus, test.expectedStatus)
		})
	}
}

func testGetInitializationStatus(t *testing.T, serverReturnedStatus InitStatus, expectedStatus InitStatus) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, serverReturnedStatus.String())
	}
	testServer = httptest.NewServer(http.HandlerFunc(handler))
	testClient = testServer.Client()
	httpClient, _ = NewTestClient(testClient, testServer.URL, DefaultEtcdConfigFilePath)
	req, _ := httpClient.GetInitializationStatus()
	if req != expectedStatus {
		t.Errorf("Wrong status read %s. Expected %s -> %s", req.String(), expectedStatus.String(), serverReturnedStatus.String())
	}
	testServer.Close()
}
