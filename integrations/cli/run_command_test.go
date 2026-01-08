// +build integration,scenarios

package cli

import (
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestRunCommandBasic verifies `coi run` command execution
//
// Scenarios covered:
// - Run simple echo command
// - Verify output captured
// - Verify exit code success
func TestRunCommandBasic(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("run echo command", func(t *testing.T) {
		result := fixture.Setup(t)

		output := testutil.AssertCommandSucceeds(t, result.Manager, "echo 'hello world'")

		expected := "hello world\n"
		if output != expected {
			t.Errorf("Expected %q, got %q", expected, output)
		}

		_ = fixture.Cleanup(t, false)
	})
}

// TestRunCommandWithCapture verifies output capture
//
// Scenarios covered:
// - Capture stdout
// - Verify captured output matches expected
func TestRunCommandWithCapture(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("capture output", func(t *testing.T) {
		result := fixture.Setup(t)

		opts := container.ExecCommandOptions{Capture: true}
		output, err := result.Manager.ExecCommand("echo 'test output'", opts)

		testutil.AssertNoError(t, err)

		expected := "test output\n"
		if output != expected {
			t.Errorf("Expected %q, got %q", expected, output)
		}

		_ = fixture.Cleanup(t, false)
	})
}

// TestRunCommandFailed verifies failed command handling
//
// Scenarios covered:
// - Run command that fails
// - Verify error returned
// - Verify non-zero exit code
func TestRunCommandFailed(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("failed command", func(t *testing.T) {
		result := fixture.Setup(t)

		// Run command that will fail
		testutil.AssertCommandFails(t, result.Manager, "exit 1")

		_ = fixture.Cleanup(t, false)
	})
}

// TestRunCommandWithWorkspace verifies workspace access
//
// Scenarios covered:
// - Run command that accesses workspace
// - Verify workspace files accessible
func TestRunCommandWithWorkspace(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	// Create workspace with test file
	workspaceFixture := testutil.NewWorkspaceFixture(t).
		WithFile("test-data.txt", "test data content")

	workspace := workspaceFixture.Create(t)

	fixture := testutil.NewSessionFixture(t)
	fixture.Workspace = workspace

	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("access workspace files", func(t *testing.T) {
		result := fixture.Setup(t)

		// Run command that reads workspace file
		output := testutil.AssertCommandSucceeds(t, result.Manager, "cat /workspace/test-data.txt")

		expected := "test data content"
		if output != expected {
			t.Errorf("Expected %q, got %q", expected, output)
		}

		_ = fixture.Cleanup(t, false)
	})
}
