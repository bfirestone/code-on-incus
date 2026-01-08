// +build integration

package container_test

import (
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

func TestManagerIntegration(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	containerName := "test-manager-integration"
	defer testutil.CleanupTestContainers(t, containerName)

	mgr := container.NewManager(containerName)

	t.Run("Launch and delete ephemeral container", func(t *testing.T) {
		// Launch
		if err := mgr.Launch("ubuntu/22.04", true); err != nil {
			t.Fatalf("Failed to launch container: %v", err)
		}

		// Verify running
		running, err := mgr.Running()
		if err != nil {
			t.Fatalf("Failed to check if running: %v", err)
		}
		if !running {
			t.Error("Container should be running")
		}

		// Stop
		if err := mgr.Stop(true); err != nil {
			t.Fatalf("Failed to stop container: %v", err)
		}

		// Delete
		if err := mgr.Delete(true); err != nil {
			t.Fatalf("Failed to delete container: %v", err)
		}
	})

	t.Run("Execute command in container", func(t *testing.T) {
		// Launch
		if err := mgr.Launch("ubuntu/22.04", true); err != nil {
			t.Fatalf("Failed to launch container: %v", err)
		}
		defer mgr.Delete(true)

		// Execute echo command
		opts := container.ExecCommandOptions{Capture: true}
		output, err := mgr.ExecCommand("echo 'hello world'", opts)
		if err != nil {
			t.Fatalf("Failed to execute command: %v", err)
		}

		if output != "hello world\n" {
			t.Errorf("Expected 'hello world\\n', got %q", output)
		}
	})

	t.Run("Mount workspace", func(t *testing.T) {
		workspace := testutil.CreateTestWorkspace(t)

		// Launch
		if err := mgr.Launch("ubuntu/22.04", true); err != nil {
			t.Fatalf("Failed to launch container: %v", err)
		}
		defer mgr.Delete(true)

		// Mount workspace
		if err := mgr.MountDisk("workspace", workspace, "/workspace", true); err != nil {
			t.Fatalf("Failed to mount workspace: %v", err)
		}

		// Verify mount by reading test file
		opts := container.ExecCommandOptions{Capture: true}
		output, err := mgr.ExecCommand("cat /workspace/test.txt", opts)
		if err != nil {
			t.Fatalf("Failed to read mounted file: %v", err)
		}

		if output != "test content" {
			t.Errorf("Expected 'test content', got %q", output)
		}
	})

	t.Run("Create and read file in container", func(t *testing.T) {
		// Launch
		if err := mgr.Launch("ubuntu/22.04", true); err != nil {
			t.Fatalf("Failed to launch container: %v", err)
		}
		defer mgr.Delete(true)

		// Create file
		content := "test file content"
		if err := mgr.CreateFile("/tmp/testfile.txt", content); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Read it back
		opts := container.ExecCommandOptions{Capture: true}
		output, err := mgr.ExecCommand("cat /tmp/testfile.txt", opts)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if output != content {
			t.Errorf("Expected %q, got %q", content, output)
		}
	})
}
