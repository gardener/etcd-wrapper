package bootstrap

import (
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/gardener/etcd-wrapper/internal/brclient"
)

var (
	pwd, _           = os.Getwd()
	exitCodeFilePath = pwd + "/../../test/exit_code"
)

func TestCleanupExitCode(t *testing.T) {
	// 1) When exit_code file exists
	// Create exit_code file
	interrupt := []byte("test")
	err := os.WriteFile(exitCodeFilePath, interrupt, 0644)
	if err != nil {
		t.Errorf("Error creating exit_code file: %v", err)
	}
	// Call CleanupExitCode
	err = CleanupExitCode(exitCodeFilePath)
	if err != nil {
		t.Errorf("Error cleaning exit_code file: %v", err)
	}
	// Check if exit_code file exists
	_, err = os.Stat(exitCodeFilePath)
	if err == nil {
		t.Errorf("exit_code file still exists after CleanupExitCode() succeeds: %v", err)
	}

	// 2) When exit_code file does not exist
	//Remove exit_code file if it exists
	_, err = os.Stat(exitCodeFilePath)
	if err != nil {
		_ = os.Remove(exitCodeFilePath)
	}
	err = CleanupExitCode(exitCodeFilePath)
	if err != nil {
		t.Error("CleanupExitCode should not throw an error")
	}
}

func TestCaptureExitCode(t *testing.T) {
	//Test with os.Interrupt
	CaptureExitCode(os.Interrupt, exitCodeFilePath)
	//Check exit_code file
	fileDataBytes, err := os.ReadFile(exitCodeFilePath)
	if err != nil {
		t.Errorf("Error reading exit_code file: %v", err)
	}
	fileData := strings.TrimSpace(string(fileDataBytes))
	if fileData != os.Interrupt.String() {
		t.Error("Wrong exit_code captured")
	}
	//Delete exit_code file
	err = os.Remove(exitCodeFilePath)
	if err != nil {
		t.Errorf("Error deleting exit_code file %v", err)
	}

	//Test with SIGTERM signal
	CaptureExitCode(syscall.SIGTERM, exitCodeFilePath)
	//Check exit_code file
	fileDataBytes, err = os.ReadFile(exitCodeFilePath)
	if err != nil {
		t.Errorf("Error reading exit_code file: %v", err)
	}
	fileData = strings.TrimSpace(string(fileDataBytes))
	if fileData != syscall.SIGTERM.String() {
		t.Error("Wrong exit_code captured")
	}
	//Delete exit_code file
	err = os.Remove(exitCodeFilePath)
	if err != nil {
		t.Errorf("Error deleting exit_code file %v", err)
	}
}

func TestGetValidationMode(t *testing.T) {
	tests := []struct {
		name                   string
		description            string
		exitCode               string
		expectedValidationMode brclient.ValidationType
		run                    func(t *testing.T, exitCode string, expectedValidationMode brclient.ValidationType)
	}{
		{"NoExitCode", "Exit code file does not exist", "", brclient.FullValidation, testValidationMode},
		{"Interrupt", "Exit code has error string interrupt", os.Interrupt.String(), brclient.SanityValidation, testValidationMode},
		{"Terminated", "Exit code has error string terminated", syscall.SIGTERM.String(), brclient.SanityValidation, testValidationMode},
		{"Random", "Exit code has any other error string", "test", brclient.FullValidation, testValidationMode},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			test.run(t, test.exitCode, test.expectedValidationMode)
		})
	}
}

func testValidationMode(t *testing.T, exitCode string, expectedValidationMode brclient.ValidationType) {
	// Create exit_code file
	if exitCode != "" {
		err := os.WriteFile(exitCodeFilePath, []byte(exitCode), 0644)
		if err != nil {
			t.Errorf("Error writing to exit_code file: %v", err)
		}
	}
	validationMode := getValidationMode(exitCodeFilePath)
	if validationMode != expectedValidationMode {
		t.Errorf("Wrong Validation mode. Expected %v, but got %v", expectedValidationMode, validationMode)
	}
	// Delete exit_code file
	if exitCode != "" {
		err := os.Remove(exitCodeFilePath)
		if err != nil {
			t.Errorf("Error removing exit_code file: %v", err)
		}
	}
}
