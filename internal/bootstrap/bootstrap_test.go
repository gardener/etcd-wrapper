// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/gardener/etcd-wrapper/internal/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/gardener/etcd-wrapper/internal/brclient"
	. "github.com/onsi/gomega"
)

type TestRoundTripper func(req *http.Request) *http.Response

func (f TestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestChangeFilePermissions(t *testing.T) {
	table := []struct {
		description string
		setup       func(testDir string) string
		mode        os.FileMode
		expectError bool
		verify      func(t *testing.T, testDir string, mode os.FileMode)
	}{
		{
			description: "change permissions of files in a directory recursively",
			setup: func(testDir string) string {
				filePath := filepath.Join(testDir, "testfile")
				err := os.WriteFile(filePath, []byte("test"), 0644)
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				subDirPath := filepath.Join(testDir, "subdir")
				err = os.Mkdir(subDirPath, 0755)
				if err != nil {
					t.Fatalf("failed to create test directory: %v", err)
				}
				subFilePath := filepath.Join(subDirPath, "subfile")
				err = os.WriteFile(subFilePath, []byte("test"), 0644)
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return testDir
			},
			mode:        0600,
			expectError: false,
			verify: func(t *testing.T, testDir string, mode os.FileMode) {
				g := NewWithT(t)
				paths := []string{
					filepath.Join(testDir, "testfile"),
					filepath.Join(testDir, "subdir", "subfile"),
				}
				for _, path := range paths {
					info, err := os.Stat(path)
					g.Expect(err).To(BeNil())
					g.Expect(info.Mode().Perm()).To(Equal(mode))
				}
			},
		},
		{
			description: "return nil error for non-existent directory",
			setup: func(testDir string) string {
				return filepath.Join(testDir, "nonexistent/path")
			},
			mode:        0600,
			expectError: false,
			verify:      func(_ *testing.T, _ string, _ os.FileMode) {},
		},
		{
			description: "return error when path is a file, not a directory",
			setup: func(testDir string) string {
				filePath := filepath.Join(testDir, "testfile")
				err := os.WriteFile(filePath, []byte("test"), 0644)
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return filePath
			},
			mode:        0600,
			expectError: true,
			verify:      func(_ *testing.T, _ string, _ os.FileMode) {},
		},
	}

	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			g := NewWithT(t)
			testDir := createTestDir(t)
			defer deleteTestDir(t, testDir)

			path := entry.setup(testDir)
			err := ChangeFilePermissions(path, entry.mode)
			g.Expect(err != nil).To(Equal(entry.expectError))

			entry.verify(t, testDir, entry.mode)
		})
	}
}

func TestCleanupExitCodeFile(t *testing.T) {
	table := []struct {
		description string
		testFn      func(*testing.T, string)
	}{
		{"cleanup exit code file when file exists", testCleanupExitCodeWhenFileExists},
		{"cleanup exit code file when it does not exist", testCleanupExitCodeWhenFileDoesNotExist},
	}

	for _, entry := range table {
		testDir := createTestDir(t)
		exitCodeFilePath := filepath.Join(testDir, "exit_code")
		t.Run(entry.description, func(t *testing.T) {
			defer deleteTestDir(t, testDir)
			entry.testFn(t, exitCodeFilePath)
		})
	}
}

func testCleanupExitCodeWhenFileExists(t *testing.T, exitCodeFilePath string) {
	g := NewWithT(t)

	// create exit code file
	interrupt := []byte("testutil")
	err := os.WriteFile(exitCodeFilePath, interrupt, 0644)
	g.Expect(err).To(BeNil())
	// Call CleanupExitCode
	err = CleanupExitCode(exitCodeFilePath)
	g.Expect(err).To(BeNil())
	// Check if exit_code file exists
	_, err = os.Stat(exitCodeFilePath)
	g.Expect(err).ToNot(BeNil())
}

func testCleanupExitCodeWhenFileDoesNotExist(t *testing.T, exitCodeFilePath string) {
	g := NewWithT(t)
	// check is exit_code file exists
	// remove it if it does
	_, err := os.Stat(exitCodeFilePath)
	if err != nil {
		_ = os.Remove(exitCodeFilePath)
	}
	// call CleanupExitCode
	err = CleanupExitCode(exitCodeFilePath)
	g.Expect(err).To(BeNil())
}

func TestCaptureExitCode(t *testing.T) {

	table := []struct {
		description             string
		signal                  os.Signal
		fileExpectedToBeCreated bool
		expectedExitCode        string
		expectError             bool
	}{
		{"do nothing when signal is nil", nil, false, "", false},
		{"capture signal in exit code when it is not nil", os.Interrupt, true, os.Interrupt.String(), false},
		{"return error when WriteFile fails", os.Interrupt, false, os.Interrupt.String(), true},
	}

	for _, entry := range table {
		testDir := createTestDir(t)
		var exitCodeFilePath string
		if entry.expectError {
			exitCodeFilePath = filepath.Join(testDir, "folder/exit_code")
		} else {
			exitCodeFilePath = filepath.Join(testDir, "exit_code")
		}
		t.Run(entry.description, func(t *testing.T) {
			g := NewWithT(t)
			defer deleteTestDir(t, testDir)
			g.Expect(CaptureExitCode(entry.signal, exitCodeFilePath) != nil).To(Equal(entry.expectError))
			if _, err := os.Stat(exitCodeFilePath); err != nil {
				notFoundError := os.IsNotExist(err)
				g.Expect(entry.fileExpectedToBeCreated).ToNot(Equal(notFoundError))
			} else {
				fileDataBytes, err := os.ReadFile(exitCodeFilePath)
				g.Expect(err).To(BeNil())
				g.Expect(entry.expectedExitCode).To(Equal(string(fileDataBytes)))
			}
		})
	}
}

func TestGetValidationMode(t *testing.T) {
	logger := zaptest.NewLogger(t)
	table := []struct {
		description            string
		exitCode               string
		expectedValidationMode brclient.ValidationType
	}{
		{"exit code file not being present should result in full validation", "", brclient.FullValidation},
		{"exit code having error string `interrupt` should result in sanity validation", os.Interrupt.String(), brclient.SanityValidation},
		{"exit code having error string `terminated` should result in sanity validation", syscall.SIGTERM.String(), brclient.SanityValidation},
		{"exit code having any other error string should result in full validation", "testutil", brclient.FullValidation},
	}
	for _, entry := range table {
		testDir := createTestDir(t)
		exitCodeFilePath := filepath.Join(testDir, "exit_code")
		t.Run(entry.description, func(t *testing.T) {
			g := NewWithT(t)
			t.Log(entry.description)

			defer deleteTestDir(t, testDir)

			// Create exit_code file
			if entry.exitCode != "" {
				err := os.WriteFile(exitCodeFilePath, []byte(entry.exitCode), 0644)
				g.Expect(err).To(BeNil())
			}
			validationMode := determineValidationMode(exitCodeFilePath, logger)
			g.Expect(validationMode).To(Equal(entry.expectedValidationMode))
		})
	}
}

func TestTryGetEtcdConfig(t *testing.T) {
	table := []struct {
		description        string
		serverReturnsError bool
		expectError        bool
	}{
		{"should not return error when etcd config is returned", false, false},
		{"should return error when invalid etcd config is returned", true, true},
	}
	for _, entry := range table {
		testDir := createTestDir(t)
		etcdConfigFilePath := filepath.Join(testDir, "etcdConfig.yaml")
		t.Run(entry.description, func(t *testing.T) {
			g := NewWithT(t)
			t.Log(entry.description)

			var httpClient *http.Client
			defer deleteTestDir(t, testDir)

			if entry.serverReturnsError {
				httpClient = getTestHttpClient(http.StatusNotFound, []byte("invalid config"))
			} else {
				httpClient = getTestHttpClient(http.StatusOK, []byte(""))
			}

			brc := brclient.NewClient(httpClient, "", etcdConfigFilePath)

			loggerConfig := zap.NewDevelopmentConfig()
			lgr, err := loggerConfig.Build()
			g.Expect(err).ToNot(HaveOccurred())

			i := initializer{brClient: brc, logger: lgr}
			_, err = i.tryGetEtcdConfig(context.TODO(), 5, time.Second)
			g.Expect(err != nil).To(Equal(entry.expectError))
		})
	}
}

func TestNewEtcdInitializer(t *testing.T) {
	table := []struct {
		description   string
		sidecarConfig types.BackupRestoreConfig
		expectError   bool
	}{
		{"should return error when invalid sidecar config is passed", createSidecarConfig(true, "", ""), true},
		{"should return error when br client creation fails", createSidecarConfig(true, ":2379", "/wrong/file/path"), true},
		{"should not return error when sidecar config is valid and br client creation succeeds", createSidecarConfig(false, ":2379", ""), false},
	}

	for _, entry := range table {
		t.Run(entry.description, func(t *testing.T) {
			g := NewWithT(t)
			t.Log(entry.description)

			loggerConfig := zap.NewDevelopmentConfig()
			lgr, err := loggerConfig.Build()
			g.Expect(err).ToNot(HaveOccurred())

			_, err = NewEtcdInitializer(&entry.sidecarConfig, lgr)
			g.Expect(err != nil).To(Equal(entry.expectError))
		})
	}
}

func createTestDir(t *testing.T) string {
	g := NewWithT(t)
	testDir, err := os.MkdirTemp("", "etcd-wrapper")
	g.Expect(err).To(BeNil())
	return testDir
}

func deleteTestDir(t *testing.T, testDir string) {
	g := NewWithT(t)
	if _, err := os.Stat(testDir); err == nil {
		err = os.RemoveAll(testDir)
		g.Expect(err).To(BeNil())
	}
}

func getTestHttpClient(responseCode int, responseBody []byte) *http.Client {
	return &http.Client{
		Transport: TestRoundTripper(func(_ *http.Request) *http.Response {
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

func createSidecarConfig(tlsEnabled bool, hostPort string, caCertBundlePath string) types.BackupRestoreConfig {
	return types.BackupRestoreConfig{
		HostPort:         hostPort,
		TLSEnabled:       tlsEnabled,
		CaCertBundlePath: caCertBundlePath,
	}
}
