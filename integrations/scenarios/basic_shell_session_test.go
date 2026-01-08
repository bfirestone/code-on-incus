// +build integration,scenarios

package scenarios

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/session"
	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestBasicShellSession verifies basic shell session creation and execution
//
// Scenarios covered:
// - Launch session in default (sandbox) mode
// - Execute simple command in container
// - Verify workspace mounted correctly
// - Verify container runs as claude user (UID 1000)
// - Verify .claude directory created
// - Clean exit and container cleanup
func TestBasicShellSession(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	var result *session.SetupResult

	t.Run("launch session in default mode", func(t *testing.T) {
		result = fixture.Setup(t)

		testutil.AssertContainerRunning(t, result.Manager)

		if result.Image != session.SandboxImage {
			t.Errorf("Expected sandbox image %q, got %q", session.SandboxImage, result.Image)
		}

		if result.ContainerName != fixture.ContainerName {
			t.Errorf("Expected container name %q, got %q", fixture.ContainerName, result.ContainerName)
		}
	})

	t.Run("execute simple command", func(t *testing.T) {
		if result == nil {
			t.Skip("Setup did not complete successfully")
		}

		expected := "hello world\n"
		output := testutil.AssertCommandSucceeds(t, result.Manager, "echo 'hello world'")

		if output != expected {
			t.Errorf("Expected output %q, got %q", expected, output)
		}
	})

	t.Run("verify workspace mounted", func(t *testing.T) {
		if result == nil {
			t.Skip("Setup did not complete successfully")
		}

		// Verify /workspace directory exists
		testutil.AssertDirExists(t, result.Manager, "/workspace")

		// Verify test file from workspace is accessible
		testutil.AssertFileExists(t, result.Manager, "/workspace/test.txt")

		// Verify content matches
		testutil.AssertFileContent(t, result.Manager, "/workspace/test.txt", "test content")
	})

	t.Run("verify container runs as claude user", func(t *testing.T) {
		if result == nil {
			t.Skip("Setup did not complete successfully")
		}

		// Check current user is claude
		output := testutil.AssertCommandSucceeds(t, result.Manager, "whoami")
		expected := "claude\n"

		if output != expected {
			t.Errorf("Expected user %q, got %q", "claude", output)
		}

		// Check UID is 1000
		output = testutil.AssertCommandSucceeds(t, result.Manager, "id -u")
		expected = "1000\n"

		if output != expected {
			t.Errorf("Expected UID %q, got %q", "1000", output)
		}
	})

	t.Run("verify .claude directory created", func(t *testing.T) {
		if result == nil {
			t.Skip("Setup did not complete successfully")
		}

		// Verify .claude directory exists in home
		claudeDir := filepath.Join(result.HomeDir, ".claude")
		testutil.AssertDirExists(t, result.Manager, claudeDir)
	})

	t.Run("clean exit and container cleanup", func(t *testing.T) {
		if result == nil {
			t.Skip("Setup did not complete successfully")
		}

		// Cleanup with session save
		err := fixture.Cleanup(t, true)
		testutil.AssertNoError(t, err)

		// Verify container is stopped/deleted (ephemeral)
		testutil.AssertContainerNotRunning(t, result.Manager)

		// Verify session data was saved
		testutil.AssertSessionSaved(t, fixture.SessionsDir, fixture.SessionID)

		// Verify .claude directory exists on host
		savedClaudePath := filepath.Join(fixture.SessionsDir, fixture.SessionID, ".claude")
		testutil.AssertFileOnHost(t, savedClaudePath)
	})
}

// TestBasicShellSessionWithFiles verifies file operations in shell session
//
// Scenarios covered:
// - Create file in workspace from host
// - Read file from container
// - Write file from container
// - Verify file appears on host
// - Verify UID shifting works (file ownership correct on host)
func TestBasicShellSessionWithFiles(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	// Create workspace with additional test file
	workspaceFixture := testutil.NewWorkspaceFixture(t).
		WithFile("input.txt", "input content").
		WithFile("subdir/nested.txt", "nested content")

	workspacePath := workspaceFixture.Create(t)

	// Create session fixture with custom workspace
	fixture := testutil.NewSessionFixture(t)
	fixture.Workspace = workspacePath

	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	result := fixture.Setup(t)

	t.Run("read files from host in container", func(t *testing.T) {
		// Verify input.txt exists and has correct content
		testutil.AssertFileContent(t, result.Manager, "/workspace/input.txt", "input content")

		// Verify nested file exists and has correct content
		testutil.AssertFileContent(t, result.Manager, "/workspace/subdir/nested.txt", "nested content")
	})

	t.Run("write file from container to workspace", func(t *testing.T) {
		// Create file in container
		content := "created by container"
		_, err := result.Manager.ExecCommand(
			"echo '"+content+"' > /workspace/output.txt",
			container.ExecCommandOptions{Capture: true},
		)
		testutil.AssertNoError(t, err)

		// Verify file exists on host
		outputPath := filepath.Join(workspacePath, "output.txt")
		testutil.AssertFileOnHost(t, outputPath)

		// Verify content matches (with extra newline from echo)
		expected := content + "\n"
		testutil.AssertFileContentOnHost(t, outputPath, expected)
	})

	t.Run("verify UID shifting", func(t *testing.T) {
		// Create file as claude user (UID 1000 in container)
		_, err := result.Manager.ExecCommand(
			"touch /workspace/test-ownership.txt",
			container.ExecCommandOptions{Capture: true},
		)
		testutil.AssertNoError(t, err)

		// Check file ownership on host
		testFilePath := filepath.Join(workspacePath, "test-ownership.txt")
		info, err := os.Stat(testFilePath)
		testutil.AssertNoError(t, err)

		// File should be owned by current user on host, not root
		// This verifies UID shifting is working
		currentUID := os.Getuid()
		fileUID := int(info.Sys().(*syscall.Stat_t).Uid)

		if fileUID != currentUID {
			t.Errorf("UID shifting not working: file UID %d != current UID %d", fileUID, currentUID)
		}
	})

	t.Run("cleanup", func(t *testing.T) {
		err := fixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)
	})
}
