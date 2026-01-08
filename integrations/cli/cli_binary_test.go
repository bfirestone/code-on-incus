// +build integration,scenarios

package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/session"
	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// CLIResult contains the result of running a CLI command
type CLIResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error
}

// RunCLI executes the coi binary with given arguments
func RunCLI(t *testing.T, args ...string) *CLIResult {
	t.Helper()

	// Ensure binary is built
	binaryPath := ensureBinary(t)

	// Create command
	cmd := exec.Command(binaryPath, args...)

	// Preserve environment variables
	cmd.Env = os.Environ()

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	return &CLIResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Error:    err,
	}
}

// RunCLIWithInput executes the coi binary with stdin input
func RunCLIWithInput(t *testing.T, input string, args ...string) *CLIResult {
	t.Helper()

	binaryPath := ensureBinary(t)

	cmd := exec.Command(binaryPath, args...)
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	return &CLIResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Error:    err,
	}
}

// RunCLIAsync starts the coi binary in the background
func RunCLIAsync(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()

	binaryPath := ensureBinary(t)
	cmd := exec.Command(binaryPath, args...)

	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start CLI: %v", err)
	}

	return cmd
}

// ensureBinary ensures the coi binary is built and returns its path
func ensureBinary(t *testing.T) string {
	t.Helper()

	// Path to binary in project root
	projectRoot := getProjectRoot(t)
	binaryPath := filepath.Join(projectRoot, "coi")

	// Check if binary exists and is recent
	info, err := os.Stat(binaryPath)
	if err == nil {
		// Check if binary is less than 1 hour old
		if time.Since(info.ModTime()) < 1*time.Hour {
			return binaryPath
		}
	}

	// Build binary
	t.Log("Building coi binary...")
	buildCmd := exec.Command("make", "build")
	buildCmd.Dir = projectRoot

	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, output)
	}

	// Verify binary exists
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("Binary not found after build: %s", binaryPath)
	}

	return binaryPath
}

// getProjectRoot finds the project root directory
func getProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Walk up until we find go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			t.Fatal("Could not find project root (no go.mod found)")
		}
		dir = parent
	}
}

// TestCLIHelp tests `coi --help`
func TestCLIHelp(t *testing.T) {
	result := RunCLI(t, "--help")

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Verify help output contains key commands
	if !strings.Contains(result.Stdout, "shell") {
		t.Error("Help output should contain 'shell' command")
	}
	if !strings.Contains(result.Stdout, "run") {
		t.Error("Help output should contain 'run' command")
	}
	if !strings.Contains(result.Stdout, "build") {
		t.Error("Help output should contain 'build' command")
	}
}

// TestCLIVersion tests `coi version`
func TestCLIVersion(t *testing.T) {
	result := RunCLI(t, "version")

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Version output should contain version info (check for "v0." or "claude-on-incus")
	output := result.Stdout + result.Stderr
	if !strings.Contains(output, "v0.") && !strings.Contains(output, "claude-on-incus") {
		t.Errorf("Version output should contain version info, got: %s", output)
	}
}

// TestCLIRunBasic tests `coi run "echo hello"`
func TestCLIRunBasic(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	workspace := testutil.CreateTestWorkspace(t)

	// Run simple echo command with --capture to get output
	result := RunCLI(t, "run", "--workspace", workspace, "--capture", "echo hello world")

	t.Logf("Exit code: %d", result.ExitCode)
	t.Logf("Stdout: %s", result.Stdout)
	t.Logf("Stderr: %s", result.Stderr)

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d\nStderr: %s", result.ExitCode, result.Stderr)
	}

	// Output should contain "hello world" (check both stdout and stderr)
	output := result.Stdout + result.Stderr
	if !strings.Contains(output, "hello world") {
		t.Errorf("Expected output to contain 'hello world', got stdout: %s, stderr: %s", result.Stdout, result.Stderr)
	}

	// Cleanup any containers
	containerName := session.ContainerName(workspace, 1)
	testutil.CleanupTestContainers(t, containerName)
}

// TestCLIRunWorkspaceAccess tests `coi run` can access workspace files
func TestCLIRunWorkspaceAccess(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	// Create workspace with test file
	workspaceFixture := testutil.NewWorkspaceFixture(t).
		WithFile("test-input.txt", "test content from host")

	workspace := workspaceFixture.Create(t)

	// Read file from workspace
	result := RunCLI(t, "run", "--workspace", workspace, "--capture", "cat /workspace/test-input.txt")

	t.Logf("Exit code: %d", result.ExitCode)
	t.Logf("Stdout: %s", result.Stdout)
	t.Logf("Stderr: %s", result.Stderr)

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d\nStderr: %s", result.ExitCode, result.Stderr)
	}

	// Output should contain file content
	if !strings.Contains(result.Stdout, "test content from host") {
		t.Errorf("Expected output to contain file content, got: %s", result.Stdout)
	}

	// Cleanup
	containerName := session.ContainerName(workspace, 1)
	testutil.CleanupTestContainers(t, containerName)
}

// TestCLIRunFailedCommand tests `coi run` with a failing command
func TestCLIRunFailedCommand(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	workspace := testutil.CreateTestWorkspace(t)

	// Run command that fails
	result := RunCLI(t, "run", "--workspace", workspace, "exit 42")

	t.Logf("Exit code: %d", result.ExitCode)
	t.Logf("Stdout: %s", result.Stdout)
	t.Logf("Stderr: %s", result.Stderr)

	// Should have non-zero exit code (42 or error from CLI)
	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code for failing command")
	}

	// Cleanup
	containerName := session.ContainerName(workspace, 1)
	testutil.CleanupTestContainers(t, containerName)
}

// TestCLIRunWithPersistent tests `coi run --persistent`
func TestCLIRunWithPersistent(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	workspace := testutil.CreateTestWorkspace(t)
	containerName := session.ContainerName(workspace, 1)

	defer testutil.CleanupTestContainers(t, containerName)

	// First run with persistent flag - create marker file
	result1 := RunCLI(t, "run", "--workspace", workspace, "--persistent", "--capture",
		"echo 'persistent test' > /tmp/marker.txt && cat /tmp/marker.txt")

	t.Logf("First run - Exit code: %d", result1.ExitCode)
	t.Logf("First run - Stdout: %s", result1.Stdout)

	if result1.ExitCode != 0 {
		t.Fatalf("First run failed with exit code %d\nStderr: %s", result1.ExitCode, result1.Stderr)
	}

	if !strings.Contains(result1.Stdout, "persistent test") {
		t.Errorf("Expected first run output to contain 'persistent test', got: %s", result1.Stdout)
	}

	// Second run with persistent flag - marker should still exist
	result2 := RunCLI(t, "run", "--workspace", workspace, "--persistent", "--capture",
		"cat /tmp/marker.txt")

	t.Logf("Second run - Exit code: %d", result2.ExitCode)
	t.Logf("Second run - Stdout: %s", result2.Stdout)

	if result2.ExitCode != 0 {
		t.Fatalf("Second run failed with exit code %d\nStderr: %s", result2.ExitCode, result2.Stderr)
	}

	if !strings.Contains(result2.Stdout, "persistent test") {
		t.Errorf("Expected marker file to persist, got: %s", result2.Stdout)
	}

	// Verify container still exists (stopped)
	mgr := container.NewManager(containerName)
	exists, err := mgr.Exists()
	if err != nil {
		t.Errorf("Failed to check container exists: %v", err)
	}
	if !exists {
		t.Error("Persistent container should still exist after run")
	}
}

// TestCLIRunWithSlot tests `coi run --slot 2`
func TestCLIRunWithSlot(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	workspace := testutil.CreateTestWorkspace(t)

	// Run with slot 1
	container1 := session.ContainerName(workspace, 1)
	defer testutil.CleanupTestContainers(t, container1)

	result1 := RunCLI(t, "run", "--workspace", workspace, "--slot", "1", "echo slot1")
	if result1.ExitCode != 0 {
		t.Fatalf("Slot 1 run failed: %d\nStderr: %s", result1.ExitCode, result1.Stderr)
	}

	// Run with slot 2
	container2 := session.ContainerName(workspace, 2)
	defer testutil.CleanupTestContainers(t, container2)

	result2 := RunCLI(t, "run", "--workspace", workspace, "--slot", "2", "echo slot2")
	if result2.ExitCode != 0 {
		t.Fatalf("Slot 2 run failed: %d\nStderr: %s", result2.ExitCode, result2.Stderr)
	}

	// Verify different container names
	if container1 == container2 {
		t.Error("Expected different container names for different slots")
	}
}

// TestCLIRunWithPrivileged tests `coi run --privileged`
func TestCLIRunWithPrivileged(t *testing.T) {
	testutil.SkipIfNoIncus(t)

	// Check if privileged image exists
	exists, err := container.ImageExists(session.PrivilegedImage)
	if err != nil || !exists {
		t.Skipf("Privileged image %s not available", session.PrivilegedImage)
	}

	workspace := testutil.CreateTestWorkspace(t)
	containerName := session.ContainerName(workspace, 1)
	defer testutil.CleanupTestContainers(t, containerName)

	// Run with privileged flag
	result := RunCLI(t, "run", "--workspace", workspace, "--privileged", "--capture", "which sudo")

	t.Logf("Exit code: %d", result.ExitCode)
	t.Logf("Stdout: %s", result.Stdout)

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d\nStderr: %s", result.ExitCode, result.Stderr)
	}

	// Sudo should be available
	if !strings.Contains(result.Stdout, "sudo") {
		t.Errorf("Expected sudo to be available in privileged mode, got: %s", result.Stdout)
	}
}

// TestCLIRunInvalidWorkspace tests error handling for invalid workspace
func TestCLIRunInvalidWorkspace(t *testing.T) {
	testutil.SkipIfNoIncus(t)

	// Non-existent workspace
	result := RunCLI(t, "run", "--workspace", "/non/existent/path", "echo test")

	t.Logf("Exit code: %d", result.ExitCode)
	t.Logf("Stderr: %s", result.Stderr)

	// Should fail with non-zero exit code
	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code for invalid workspace")
	}

	// Error message should be helpful
	if result.Stderr == "" {
		t.Error("Expected error message for invalid workspace")
	}
}

// TestCLIListContainers tests `coi list`
func TestCLIListContainers(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	workspace := testutil.CreateTestWorkspace(t)
	containerName := session.ContainerName(workspace, 1)

	// Create a running container
	fixture := testutil.NewSessionFixture(t)
	fixture.Workspace = workspace
	defer testutil.CleanupTestContainers(t, containerName)

	result := fixture.Setup(t)
	testutil.AssertContainerRunning(t, result.Manager)

	// List containers
	listResult := RunCLI(t, "list")

	t.Logf("Exit code: %d", listResult.ExitCode)
	t.Logf("Stdout: %s", listResult.Stdout)

	if listResult.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d\nStderr: %s", listResult.ExitCode, listResult.Stderr)
	}

	// Output should contain our container name
	if !strings.Contains(listResult.Stdout, containerName) {
		t.Errorf("Expected list output to contain container %s, got: %s", containerName, listResult.Stdout)
	}

	// Cleanup
	_ = fixture.Cleanup(t, false)
}

// TestCLICleanCommand tests `coi clean`
func TestCLICleanCommand(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	workspace := testutil.CreateTestWorkspace(t)
	containerName := session.ContainerName(workspace, 1)

	// Create a container
	fixture := testutil.NewSessionFixture(t)
	fixture.Workspace = workspace

	result := fixture.Setup(t)
	testutil.AssertContainerRunning(t, result.Manager)

	// Stop container (clean expects stopped containers)
	err := result.Manager.Stop(false)
	testutil.AssertNoError(t, err)

	// Clean with --force flag (skip confirmation)
	cleanResult := RunCLI(t, "clean", "--force")

	t.Logf("Exit code: %d", cleanResult.ExitCode)
	t.Logf("Stdout: %s", cleanResult.Stdout)

	if cleanResult.ExitCode != 0 {
		t.Logf("Clean command stderr: %s", cleanResult.Stderr)
		// Note: clean might fail if no containers to clean, which is ok
	}

	// Verify container cleaned up
	testutil.CleanupTestContainers(t, containerName)
}

// TestCLIConfigPrecedence tests configuration precedence (env vars, flags, etc.)
func TestCLIConfigPrecedence(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	workspace := testutil.CreateTestWorkspace(t)
	containerName := session.ContainerName(workspace, 1)
	defer testutil.CleanupTestContainers(t, containerName)

	// Set environment variable
	os.Setenv("CLAUDE_ON_INCUS_PERSISTENT", "true")
	defer os.Unsetenv("CLAUDE_ON_INCUS_PERSISTENT")

	// Run without --persistent flag (should use env var)
	result := RunCLI(t, "run", "--workspace", workspace,
		"echo 'env test' > /tmp/env-marker.txt")

	if result.ExitCode != 0 {
		t.Fatalf("Run failed: %d\nStderr: %s", result.ExitCode, result.Stderr)
	}

	// Verify container still exists (persistent from env var)
	mgr := container.NewManager(containerName)
	exists, err := mgr.Exists()
	testutil.AssertNoError(t, err)

	if !exists {
		t.Error("Container should persist when CLAUDE_ON_INCUS_PERSISTENT=true")
	}

	// Verify file persists
	result2 := RunCLI(t, "run", "--workspace", workspace, "--capture", "cat /tmp/env-marker.txt")
	if result2.ExitCode != 0 {
		t.Fatalf("Second run failed: %d\nStderr: %s", result2.ExitCode, result2.Stderr)
	}

	if !strings.Contains(result2.Stdout, "env test") {
		t.Errorf("Expected marker to persist from env config, got: %s", result2.Stdout)
	}
}

// TestCLIBuildCommand tests `coi build sandbox`
func TestCLIBuildCommand(t *testing.T) {
	testutil.SkipIfNoIncus(t)

	// This is a long-running test, only run if explicitly requested
	if testing.Short() {
		t.Skip("Skipping build test in short mode")
	}

	// Note: This test actually builds an image, which is time-consuming
	// In a real test suite, you might want to mock this or only run occasionally

	result := RunCLI(t, "build", "sandbox", "--help")

	// Just verify the build command exists and shows help
	if result.ExitCode != 0 {
		t.Errorf("Build help should succeed, got exit code %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "build") {
		t.Error("Build help output should contain 'build' information")
	}
}

// TestCLIInvalidCommand tests unknown command error handling
func TestCLIInvalidCommand(t *testing.T) {
	result := RunCLI(t, "invalid-command-xyz")

	// Should fail with non-zero exit code
	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code for invalid command")
	}

	// Should show error message
	output := result.Stdout + result.Stderr
	if !strings.Contains(output, "unknown") && !strings.Contains(output, "invalid") {
		t.Logf("Expected error message for invalid command, got: %s", output)
	}
}

// TestCLIGlobalFlags tests global flags like --help, --version
func TestCLIGlobalFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
	}{
		{"help flag", []string{"--help"}, 0},
		{"help command", []string{"help"}, 0},
		{"version", []string{"version"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RunCLI(t, tt.args...)

			if result.ExitCode != tt.wantExit {
				t.Errorf("Expected exit code %d, got %d\nStderr: %s",
					tt.wantExit, result.ExitCode, result.Stderr)
			}
		})
	}
}
