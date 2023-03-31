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

package bootstrap

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/gardener/etcd-wrapper/internal/brclient"
	. "github.com/onsi/gomega"
)

func TestCleanupExitCodeFile(t *testing.T) {
	table := []struct {
		description string
		testFn      func(*testing.T, string)
	}{
		{"cleanup exit code file when file exists", testCleanupExitCodeWhenFileExists},
		{"cleanup exit code file when it does not exist", testCleanupExitCodeWhenFileDoesNotExists},
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
	interrupt := []byte("test")
	err := os.WriteFile(exitCodeFilePath, interrupt, 0644)
	g.Expect(err).To(BeNil())
	// Call CleanupExitCode
	err = CleanupExitCode(exitCodeFilePath)
	g.Expect(err).To(BeNil())
	// Check if exit_code file exists
	_, err = os.Stat(exitCodeFilePath)
	g.Expect(err).ToNot(BeNil())
}

func testCleanupExitCodeWhenFileDoesNotExists(t *testing.T, exitCodeFilePath string) {
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
	}{
		{"do nothing when signal is nil", nil, false, ""},
		{"capture signal in exit code when it is not nil", os.Interrupt, true, os.Interrupt.String()},
	}

	for _, entry := range table {
		testDir := createTestDir(t)
		exitCodeFilePath := filepath.Join(testDir, "exit_code")
		t.Run(entry.description, func(t *testing.T) {
			g := NewWithT(t)
			defer deleteTestDir(t, testDir)
			CaptureExitCode(entry.signal, exitCodeFilePath)
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
	table := []struct {
		description            string
		exitCode               string
		expectedValidationMode brclient.ValidationType
	}{
		{"test: exit code file not being present should result in full validation", "", brclient.FullValidation},
		{"test: exit code having error string `interrupt` should result in sanity validation", os.Interrupt.String(), brclient.SanityValidation},
		{"test: exit code having error string `terminated` should result in sanity validation", syscall.SIGTERM.String(), brclient.SanityValidation},
		{"test: exit code having any other error string should result in full validation", "test", brclient.FullValidation},
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
			validationMode := getValidationMode(exitCodeFilePath)
			g.Expect(validationMode).To(Equal(entry.expectedValidationMode))
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
