package brclient

import (
	"context"
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

var (
	pwd, _             = os.Getwd()
	etcdConfigFilePath = pwd + "/../../test/etcd-config.yaml"
)

//func TestSuit(t *testing.T) {
//	allTests := []struct {
//		name   string
//		testFn func(t *testing.T)
//	}{
//		{"getEtcdConfig", testGetEtcdConfig},
//	}
//
//	for _, entry := range allTests {
//		t.Run(entry.name, entry.testFn)
//	}
//}

type TestRoundTripper func(req *http.Request) *http.Response

func (f TestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

//func testGetEtcdConfig(t *testing.T) {
//	table := []struct {
//		description  string
//		responseCode int
//		responseBody []byte
//		expectError  bool
//	}{
//		{"test: 200 response code should return a valid etcd config", http.StatusOK, []byte("give me a valid etcd config"), false},
//	}
//
//	for _, entry := range table {
//		t.Log(entry.description)
//		httpClient := &http.Client{
//			Transport: TestRoundTripper(func(req *http.Request) *http.Response {
//				var contentLen int64
//				if entry.responseBody != nil {
//					contentLen = int64(len(entry.responseBody))
//				}
//				return &http.Response{
//					StatusCode:    entry.responseCode,
//					Body:          io.NopCloser(bytes.NewReader(entry.responseBody)),
//					ContentLength: contentLen,
//				}
//			}),
//			Timeout: 5 * time.Second,
//		}
//		NewClient(httpClient, )
//	}
//}

func TestBrClient_TestGetEtcdConfig(t *testing.T) {
	_, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting present working directory: %v", err)
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "Sample etcd config")
	}
	testServer = httptest.NewServer(http.HandlerFunc(handler))
	testClient = testServer.Client()
	httpClient, err = NewClient(testClient, testServer.URL, etcdConfigFilePath)
	if err != nil {
		t.Errorf("Error creating test client: %v", err)
	}
	req, err := httpClient.GetEtcdConfig(context.TODO())
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
		{"Successful", "Initialization status returned by server is who knows", 4, InProgress, testGetInitializationStatus},
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
	httpClient, _ = NewClient(testClient, testServer.URL, DefaultEtcdConfigFilePath)
	req, _ := httpClient.GetInitializationStatus(context.TODO())
	if req != expectedStatus {
		t.Errorf("Wrong status read %s. Expected %s, received %s", req.String(), expectedStatus.String(), serverReturnedStatus.String())
	}
	testServer.Close()
}
