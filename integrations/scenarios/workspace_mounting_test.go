// +build integration,scenarios

package scenarios

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestWorkspaceMountingBasic verifies basic workspace file operations
//
// Scenarios covered:
// - Read file from host in container
// - Write file from container visible on host
// - Modify host file, visible in container
func TestWorkspaceMountingBasic(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	// Create workspace with test files
	workspaceFixture := testutil.NewWorkspaceFixture(t).
		WithFile("input.txt", "input content").
		WithFile("read-only.txt", "readonly content")

	workspace := workspaceFixture.Create(t)

	fixture := testutil.NewSessionFixture(t)
	fixture.Workspace = workspace

	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	result := fixture.Setup(t)

	t.Run("read files from host", func(t *testing.T) {
		// Verify files are accessible
		testutil.AssertFileContent(t, result.Manager, "/workspace/input.txt", "input content")
		testutil.AssertFileContent(t, result.Manager, "/workspace/read-only.txt", "readonly content")
	})

	t.Run("write file from container", func(t *testing.T) {
		// Create new file
		_, err := result.Manager.ExecCommand(
			"echo 'created by container' > /workspace/output.txt",
			testutil.ExecOpts(),
		)
		testutil.AssertNoError(t, err)

		// Verify on host
		outputPath := filepath.Join(workspace, "output.txt")
		testutil.AssertFileOnHost(t, outputPath)
		testutil.AssertFileContentOnHost(t, outputPath, "created by container\n")
	})

	t.Run("modify file from container visible on host", func(t *testing.T) {
		// Modify existing file
		_, err := result.Manager.ExecCommand(
			"echo 'modified content' > /workspace/input.txt",
			testutil.ExecOpts(),
		)
		testutil.AssertNoError(t, err)

		// Verify on host
		inputPath := filepath.Join(workspace, "input.txt")
		testutil.AssertFileContentOnHost(t, inputPath, "modified content\n")
	})

	_ = fixture.Cleanup(t, false)
}

// TestWorkspaceMountingUIDShifting verifies UID shifting works correctly
//
// Scenarios covered:
// - Verify file ownership correct on host
// - Files created by claude (UID 1000 in container) owned by current user on host
func TestWorkspaceMountingUIDShifting(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	workspace := testutil.CreateTestWorkspace(t)

	fixture := testutil.NewSessionFixture(t)
	fixture.Workspace = workspace

	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	result := fixture.Setup(t)

	t.Run("uid shifting for created files", func(t *testing.T) {
		// Create file as claude user (UID 1000 in container)
		_, err := result.Manager.ExecCommand(
			"touch /workspace/uid-test.txt",
			testutil.ExecOpts(),
		)
		testutil.AssertNoError(t, err)

		// Check file ownership on host
		testFilePath := filepath.Join(workspace, "uid-test.txt")
		info, err := os.Stat(testFilePath)
		testutil.AssertNoError(t, err)

		// File should be owned by current user on host
		currentUID := os.Getuid()
		fileUID := int(info.Sys().(*syscall.Stat_t).Uid)

		if fileUID != currentUID {
			t.Errorf("UID shifting not working: file UID %d != current UID %d", fileUID, currentUID)
		}
	})

	_ = fixture.Cleanup(t, false)
}

// TestWorkspaceMountingLargeFiles verifies large file handling
//
// Scenarios covered:
// - Create large file on host, read from container
// - Create large file in container, verify on host
func TestWorkspaceMountingLargeFiles(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	// Create workspace with large file (10MB)
	workspaceFixture := testutil.NewWorkspaceFixture(t).
		WithLargeFile("large-input.txt", 10*1024*1024)

	workspace := workspaceFixture.Create(t)

	fixture := testutil.NewSessionFixture(t)
	fixture.Workspace = workspace

	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	result := fixture.Setup(t)

	t.Run("read large file from host", func(t *testing.T) {
		// Verify large file is accessible
		testutil.AssertFileExists(t, result.Manager, "/workspace/large-input.txt")

		// Check file size
		output := testutil.AssertCommandSucceeds(t, result.Manager, "wc -c < /workspace/large-input.txt")
		expected := "10485760\n" // 10MB in bytes
		if output != expected {
			t.Errorf("Expected file size %q, got %q", expected, output)
		}
	})

	t.Run("create large file in container", func(t *testing.T) {
		// Create 5MB file
		_, err := result.Manager.ExecCommand(
			"dd if=/dev/zero of=/workspace/large-output.txt bs=1M count=5",
			testutil.ExecOpts(),
		)
		testutil.AssertNoError(t, err)

		// Verify on host
		outputPath := filepath.Join(workspace, "large-output.txt")
		info, err := os.Stat(outputPath)
		testutil.AssertNoError(t, err)

		expectedSize := int64(5 * 1024 * 1024)
		if info.Size() != expectedSize {
			t.Errorf("Expected file size %d, got %d", expectedSize, info.Size())
		}
	})

	_ = fixture.Cleanup(t, false)
}

// TestWorkspaceMountingNestedDirectories verifies nested directory handling
//
// Scenarios covered:
// - Create nested directories from container
// - Verify structure on host
func TestWorkspaceMountingNestedDirectories(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	workspace := testutil.CreateTestWorkspace(t)

	fixture := testutil.NewSessionFixture(t)
	fixture.Workspace = workspace

	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	result := fixture.Setup(t)

	t.Run("create nested directories", func(t *testing.T) {
		// Create nested structure
		_, err := result.Manager.ExecCommand(
			"mkdir -p /workspace/a/b/c/d && echo 'deep file' > /workspace/a/b/c/d/file.txt",
			testutil.ExecOpts(),
		)
		testutil.AssertNoError(t, err)

		// Verify on host
		nestedPath := filepath.Join(workspace, "a", "b", "c", "d", "file.txt")
		testutil.AssertFileOnHost(t, nestedPath)
		testutil.AssertFileContentOnHost(t, nestedPath, "deep file\n")
	})

	_ = fixture.Cleanup(t, false)
}
