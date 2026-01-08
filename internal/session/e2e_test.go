// +build integration

package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/session"
	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

func TestSessionLifecycleE2E(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	workspace := testutil.CreateTestWorkspace(t)
	sessionsDir := t.TempDir()
	containerPrefix := "test-e2e-session"

	defer testutil.CleanupTestContainers(t, containerPrefix)

	sessionID, err := session.GenerateSessionID()
	if err != nil {
		t.Fatalf("Failed to generate session ID: %v", err)
	}

	t.Run("Setup session", func(t *testing.T) {
		opts := session.SetupOptions{
			WorkspacePath: workspace,
			Privileged:    false,
			Slot:          1,
			SessionsDir:   sessionsDir,
		}

		result, err := session.Setup(opts)
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}

		if result == nil {
			t.Fatal("Setup result is nil")
		}

		if result.ContainerName == "" {
			t.Error("Container name is empty")
		}

		// Verify container is running
		running, err := result.Manager.Running()
		if err != nil {
			t.Fatalf("Failed to check if running: %v", err)
		}
		if !running {
			t.Error("Container should be running after setup")
		}

		// Create a test file in .claude directory
		claudePath := filepath.Join(result.HomeDir, ".claude", "test-state.txt")
		content := "test session state"
		if err := result.Manager.CreateFile(claudePath, content); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Cleanup
		cleanupOpts := session.CleanupOptions{
			ContainerName: result.ContainerName,
			SessionID:     sessionID,
			Privileged:    false,
			SessionsDir:   sessionsDir,
			SaveSession:   true,
		}

		if err := session.Cleanup(cleanupOpts); err != nil {
			t.Fatalf("Cleanup failed: %v", err)
		}

		// Verify session data was saved
		savedClaudePath := filepath.Join(sessionsDir, sessionID, ".claude", "test-state.txt")
		data, err := os.ReadFile(savedClaudePath)
		if err != nil {
			t.Fatalf("Failed to read saved session data: %v", err)
		}

		if string(data) != content {
			t.Errorf("Expected %q, got %q", content, string(data))
		}
	})

	t.Run("Resume session", func(t *testing.T) {
		// Setup with resume
		opts := session.SetupOptions{
			WorkspacePath: workspace,
			Privileged:    false,
			ResumeFromID:  sessionID,
			Slot:          1,
			SessionsDir:   sessionsDir,
		}

		result, err := session.Setup(opts)
		if err != nil {
			t.Fatalf("Resume setup failed: %v", err)
		}

		// Verify session data was restored
		claudePath := filepath.Join(result.HomeDir, ".claude", "test-state.txt")
		output, err := result.Manager.ExecCommand("cat "+claudePath, container.ExecCommandOptions{Capture: true})
		if err != nil {
			t.Fatalf("Failed to read restored file: %v", err)
		}

		expected := "test session state"
		if output != expected {
			t.Errorf("Expected %q, got %q", expected, output)
		}

		// Cleanup
		cleanupOpts := session.CleanupOptions{
			ContainerName: result.ContainerName,
			SessionID:     sessionID,
			Privileged:    false,
			SessionsDir:   sessionsDir,
			SaveSession:   false,
		}

		if err := session.Cleanup(cleanupOpts); err != nil {
			t.Fatalf("Cleanup failed: %v", err)
		}
	})
}

func TestMultiSlotE2E(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	workspace := testutil.CreateTestWorkspace(t)
	sessionsDir := t.TempDir()
	containerPrefix := "test-multislot"

	defer testutil.CleanupTestContainers(t, containerPrefix)

	t.Run("Run two sessions in parallel", func(t *testing.T) {
		// Setup session in slot 1
		opts1 := session.SetupOptions{
			WorkspacePath: workspace,
			Slot:          1,
			SessionsDir:   sessionsDir,
		}

		result1, err := session.Setup(opts1)
		if err != nil {
			t.Fatalf("Slot 1 setup failed: %v", err)
		}

		// Setup session in slot 2
		opts2 := session.SetupOptions{
			WorkspacePath: workspace,
			Slot:          2,
			SessionsDir:   sessionsDir,
		}

		result2, err := session.Setup(opts2)
		if err != nil {
			t.Fatalf("Slot 2 setup failed: %v", err)
		}

		// Verify both containers have different names
		if result1.ContainerName == result2.ContainerName {
			t.Error("Container names should be different for different slots")
		}

		// Verify both are running
		running1, _ := result1.Manager.Running()
		running2, _ := result2.Manager.Running()

		if !running1 || !running2 {
			t.Error("Both containers should be running")
		}

		// Cleanup both
		session.Cleanup(session.CleanupOptions{
			ContainerName: result1.ContainerName,
			SessionsDir:   sessionsDir,
		})

		session.Cleanup(session.CleanupOptions{
			ContainerName: result2.ContainerName,
			SessionsDir:   sessionsDir,
		})
	})
}

func TestWorkspaceMountingE2E(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	workspace := testutil.CreateTestWorkspace(t)
	sessionsDir := t.TempDir()
	containerPrefix := "test-workspace"

	defer testutil.CleanupTestContainers(t, containerPrefix)

	// Write a test file to workspace
	testFile := filepath.Join(workspace, "project-file.txt")
	testContent := "workspace file content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	opts := session.SetupOptions{
		WorkspacePath: workspace,
		Slot:          1,
		SessionsDir:   sessionsDir,
	}

	result, err := session.Setup(opts)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer session.Cleanup(session.CleanupOptions{
		ContainerName: result.ContainerName,
		SessionsDir:   sessionsDir,
	})

	t.Run("Read workspace file from container", func(t *testing.T) {
		output, err := result.Manager.ExecCommand("cat /workspace/project-file.txt", container.ExecCommandOptions{Capture: true})
		if err != nil {
			t.Fatalf("Failed to read workspace file: %v", err)
		}

		if output != testContent {
			t.Errorf("Expected %q, got %q", testContent, output)
		}
	})

	t.Run("Write file from container to workspace", func(t *testing.T) {
		newContent := "created by container"
		_, err := result.Manager.ExecCommand(
			"echo '"+newContent+"' > /workspace/container-created.txt",
			container.ExecCommandOptions{Capture: true},
		)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		// Read from host
		createdFile := filepath.Join(workspace, "container-created.txt")
		data, err := os.ReadFile(createdFile)
		if err != nil {
			t.Fatalf("Failed to read file from host: %v", err)
		}

		// Note: extra newline from echo
		expected := newContent + "\n"
		if string(data) != expected {
			t.Errorf("Expected %q, got %q", expected, string(data))
		}
	})
}
