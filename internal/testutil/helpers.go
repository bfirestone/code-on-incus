package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/container"
)

// ExecOpts returns default ExecCommandOptions for tests
func ExecOpts() container.ExecCommandOptions {
	return container.ExecCommandOptions{Capture: true}
}

// AssertContainerRunning verifies container is running
func AssertContainerRunning(t *testing.T, mgr *container.Manager) {
	t.Helper()

	running, err := mgr.Running()
	if err != nil {
		t.Fatalf("Failed to check if running: %v", err)
	}
	if !running {
		t.Error("Expected container to be running, but it is not")
	}
}

// AssertContainerNotRunning verifies container is not running
func AssertContainerNotRunning(t *testing.T, mgr *container.Manager) {
	t.Helper()

	running, err := mgr.Running()
	if err != nil {
		t.Fatalf("Failed to check if running: %v", err)
	}
	if running {
		t.Error("Expected container to not be running, but it is")
	}
}

// AssertFileExists verifies file exists in container
func AssertFileExists(t *testing.T, mgr *container.Manager, path string) {
	t.Helper()

	opts := container.ExecCommandOptions{Capture: true}
	_, err := mgr.ExecCommand(fmt.Sprintf("test -f %s", path), opts)
	if err != nil {
		t.Errorf("File %s does not exist in container", path)
	}
}

// AssertFileNotExists verifies file does not exist in container
func AssertFileNotExists(t *testing.T, mgr *container.Manager, path string) {
	t.Helper()

	opts := container.ExecCommandOptions{Capture: true}
	_, err := mgr.ExecCommand(fmt.Sprintf("test -f %s", path), opts)
	if err == nil {
		t.Errorf("File %s should not exist in container, but it does", path)
	}
}

// AssertDirExists verifies directory exists in container
func AssertDirExists(t *testing.T, mgr *container.Manager, path string) {
	t.Helper()

	opts := container.ExecCommandOptions{Capture: true}
	_, err := mgr.ExecCommand(fmt.Sprintf("test -d %s", path), opts)
	if err != nil {
		t.Errorf("Directory %s does not exist in container", path)
	}
}

// AssertFileContent verifies file content in container
func AssertFileContent(t *testing.T, mgr *container.Manager, path string, expected string) {
	t.Helper()

	opts := container.ExecCommandOptions{Capture: true}
	output, err := mgr.ExecCommand(fmt.Sprintf("cat %s", path), opts)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if output != expected {
		t.Errorf("File %s content mismatch:\nExpected: %q\nGot: %q", path, expected, output)
	}
}

// AssertFileContentContains verifies file content contains substring in container
func AssertFileContentContains(t *testing.T, mgr *container.Manager, path string, substring string) {
	t.Helper()

	opts := container.ExecCommandOptions{Capture: true}
	output, err := mgr.ExecCommand(fmt.Sprintf("cat %s", path), opts)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if !strings.Contains(output, substring) {
		t.Errorf("File %s does not contain expected substring:\nExpected substring: %q\nActual content: %q", path, substring, output)
	}
}

// AssertCommandSucceeds verifies command executes successfully in container
func AssertCommandSucceeds(t *testing.T, mgr *container.Manager, command string) string {
	t.Helper()

	opts := container.ExecCommandOptions{Capture: true}
	output, err := mgr.ExecCommand(command, opts)
	if err != nil {
		t.Errorf("Command failed: %s\nError: %v\nOutput: %s", command, err, output)
	}

	return output
}

// AssertCommandFails verifies command fails in container
func AssertCommandFails(t *testing.T, mgr *container.Manager, command string) {
	t.Helper()

	opts := container.ExecCommandOptions{Capture: true}
	_, err := mgr.ExecCommand(command, opts)
	if err == nil {
		t.Errorf("Command should have failed but succeeded: %s", command)
	}
}

// AssertSessionSaved verifies session data was saved to disk
func AssertSessionSaved(t *testing.T, sessionsDir, sessionID string) {
	t.Helper()

	savedPath := filepath.Join(sessionsDir, sessionID, ".claude")
	if _, err := os.Stat(savedPath); os.IsNotExist(err) {
		t.Errorf("Session data not saved at %s", savedPath)
	}
}

// AssertSessionNotSaved verifies session data was not saved to disk
func AssertSessionNotSaved(t *testing.T, sessionsDir, sessionID string) {
	t.Helper()

	savedPath := filepath.Join(sessionsDir, sessionID, ".claude")
	if _, err := os.Stat(savedPath); !os.IsNotExist(err) {
		t.Errorf("Session data should not be saved at %s, but it exists", savedPath)
	}
}

// AssertFileOnHost verifies file exists on host
func AssertFileOnHost(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("File %s does not exist on host", path)
	}
}

// AssertFileContentOnHost verifies file content on host
func AssertFileContentOnHost(t *testing.T, path string, expected string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s on host: %v", path, err)
	}

	actual := string(content)
	if actual != expected {
		t.Errorf("File %s content mismatch on host:\nExpected: %q\nGot: %q", path, expected, actual)
	}
}

// AssertFileOwnership verifies file ownership in container
func AssertFileOwnership(t *testing.T, mgr *container.Manager, path string, expectedUID, expectedGID int) {
	t.Helper()

	opts := container.ExecCommandOptions{Capture: true}
	output, err := mgr.ExecCommand(fmt.Sprintf("stat -c '%%u:%%g' %s", path), opts)
	if err != nil {
		t.Fatalf("Failed to stat file %s: %v", path, err)
	}

	expected := fmt.Sprintf("%d:%d\n", expectedUID, expectedGID)
	if output != expected {
		t.Errorf("File %s ownership mismatch:\nExpected: %s\nGot: %s", path, expected, output)
	}
}

// AssertCommandOutput verifies command output matches expected
func AssertCommandOutput(t *testing.T, mgr *container.Manager, command string, expected string) {
	t.Helper()

	opts := container.ExecCommandOptions{Capture: true}
	output, err := mgr.ExecCommand(command, opts)
	if err != nil {
		t.Fatalf("Command failed: %s\nError: %v", command, err)
	}

	if output != expected {
		t.Errorf("Command output mismatch for: %s\nExpected: %q\nGot: %q", command, expected, output)
	}
}

// AssertCommandOutputContains verifies command output contains substring
func AssertCommandOutputContains(t *testing.T, mgr *container.Manager, command string, substring string) {
	t.Helper()

	opts := container.ExecCommandOptions{Capture: true}
	output, err := mgr.ExecCommand(command, opts)
	if err != nil {
		t.Fatalf("Command failed: %s\nError: %v", command, err)
	}

	if !strings.Contains(output, substring) {
		t.Errorf("Command output does not contain expected substring:\nCommand: %s\nExpected substring: %q\nActual output: %q", command, substring, output)
	}
}

// AssertErrorContains verifies error message contains substring
func AssertErrorContains(t *testing.T, err error, substring string) {
	t.Helper()

	if err == nil {
		t.Fatalf("Expected error containing %q, but got no error", substring)
	}

	if !strings.Contains(err.Error(), substring) {
		t.Errorf("Error does not contain expected substring:\nExpected substring: %q\nActual error: %v", substring, err)
	}
}

// AssertNoError verifies there is no error
func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}
}

// AssertImageExists verifies Incus image exists
func AssertImageExists(t *testing.T, image string) {
	t.Helper()

	exists, err := container.ImageExists(image)
	if err != nil {
		t.Fatalf("Failed to check if image exists: %v", err)
	}

	if !exists {
		t.Errorf("Image %s does not exist", image)
	}
}

// AssertImageNotExists verifies Incus image does not exist
func AssertImageNotExists(t *testing.T, image string) {
	t.Helper()

	exists, err := container.ImageExists(image)
	if err != nil {
		t.Fatalf("Failed to check if image exists: %v", err)
	}

	if exists {
		t.Errorf("Image %s should not exist, but it does", image)
	}
}

// FormatCommand formats a command string with arguments
func FormatCommand(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// FormatTestName formats a test name with arguments
func FormatTestName(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
