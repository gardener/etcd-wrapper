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
	"strings"
	"syscall"
	"testing"

	"github.com/gardener/etcd-wrapper/internal/brclient"
	. "github.com/onsi/gomega"
)

var (
	pwd, _           = os.Getwd()
	exitCodeFilePath = pwd + "/../../test/exit_code"
)

func TestCleanupExitCodeWhenFileExists(t *testing.T) {
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

func TestCleanupExitCodeWhenFileDoesNotExists(t *testing.T) {
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
	g := NewWithT(t)
	//Test with os.Interrupt
	CaptureExitCode(os.Interrupt, exitCodeFilePath)

	//Check exit_code file
	fileDataBytes, err := os.ReadFile(exitCodeFilePath)
	g.Expect(err).To(BeNil())
	g.Expect(strings.TrimSpace(string(fileDataBytes))).To(Equal(os.Interrupt.String()))
	//Delete exit_code file
	err = os.Remove(exitCodeFilePath)
	g.Expect(err).To(BeNil())
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
		t.Run(entry.description, func(t *testing.T) {
			t.Log(entry.description)

			g := NewWithT(t)

			// Create exit_code file
			if entry.exitCode != "" {
				err := os.WriteFile(exitCodeFilePath, []byte(entry.exitCode), 0644)
				g.Expect(err).To(BeNil())
			}
			validationMode := getValidationMode(exitCodeFilePath)
			g.Expect(validationMode).To(Equal(entry.expectedValidationMode))
			// Cleanup -> delete exit_code file
			if entry.exitCode != "" {
				err := os.Remove(exitCodeFilePath)
				g.Expect(err).To(BeNil())
			}
		})
	}
}
