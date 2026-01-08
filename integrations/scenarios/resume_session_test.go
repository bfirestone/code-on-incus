// +build integration,scenarios

package scenarios

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/session"
	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestResumeSessionBasic verifies basic session persistence and resume
//
// Scenarios covered:
// - Create session with .claude state
// - Save session data on cleanup
// - Resume from session ID
// - Verify .claude state restored
// - Verify workspace still mounted
func TestResumeSessionBasic(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	var sessionID string
	var originalWorkspace string

	t.Run("create session and save state", func(t *testing.T) {
		result := fixture.Setup(t)
		sessionID = fixture.SessionID
		originalWorkspace = fixture.Workspace

		// Create test files in .claude directory
		claudeDir := filepath.Join(result.HomeDir, ".claude")

		// Create state file
		statePath := filepath.Join(claudeDir, "test-state.txt")
		err := result.Manager.CreateFile(statePath, "test session state")
		testutil.AssertNoError(t, err)

		// Create nested directory with file
		nestedPath := filepath.Join(claudeDir, "subdir", "nested.txt")
		_, err = result.Manager.ExecCommand("mkdir -p "+filepath.Join(claudeDir, "subdir"), testutil.ExecOpts())
		testutil.AssertNoError(t, err)
		err = result.Manager.CreateFile(nestedPath, "nested content")
		testutil.AssertNoError(t, err)

		// Verify files exist before cleanup
		testutil.AssertFileExists(t, result.Manager, statePath)
		testutil.AssertFileExists(t, result.Manager, nestedPath)

		// Cleanup with save
		err = fixture.Cleanup(t, true)
		testutil.AssertNoError(t, err)

		// Verify session saved
		testutil.AssertSessionSaved(t, fixture.SessionsDir, sessionID)

		// Verify saved files exist on host
		savedStatePath := filepath.Join(fixture.SessionsDir, sessionID, ".claude", "test-state.txt")
		testutil.AssertFileOnHost(t, savedStatePath)
		testutil.AssertFileContentOnHost(t, savedStatePath, "test session state")
	})

	t.Run("resume from session ID", func(t *testing.T) {
		// Setup with resume
		opts := session.SetupOptions{
			WorkspacePath: originalWorkspace,
			ResumeFromID:  sessionID,
			Slot:          1,
			SessionsDir:   fixture.SessionsDir,
		}

		result, err := session.Setup(opts)
		testutil.AssertNoError(t, err)

		// Verify .claude state restored
		claudeDir := filepath.Join(result.HomeDir, ".claude")

		// Verify state file restored
		statePath := filepath.Join(claudeDir, "test-state.txt")
		testutil.AssertFileContent(t, result.Manager, statePath, "test session state")

		// Verify nested file restored
		nestedPath := filepath.Join(claudeDir, "subdir", "nested.txt")
		testutil.AssertFileContent(t, result.Manager, nestedPath, "nested content")

		// Verify workspace still mounted
		testutil.AssertDirExists(t, result.Manager, "/workspace")
		testutil.AssertFileExists(t, result.Manager, "/workspace/test.txt")

		// Cleanup without save
		cleanupOpts := session.CleanupOptions{
			ContainerName: result.ContainerName,
			SessionID:     sessionID,
			SessionsDir:   fixture.SessionsDir,
			SaveSession:   false,
		}
		err = session.Cleanup(cleanupOpts)
		testutil.AssertNoError(t, err)
	})
}

// TestResumeSessionWithDifferentWorkspace verifies error when workspace changes
//
// Scenarios covered:
// - Resume with different workspace (should fail or warn)
// - Verify error message is helpful
func TestResumeSessionWithDifferentWorkspace(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	var sessionID string

	t.Run("create and save session", func(t *testing.T) {
		_ = fixture.Setup(t)
		sessionID = fixture.SessionID

		// Save session
		err := fixture.Cleanup(t, true)
		testutil.AssertNoError(t, err)
		testutil.AssertSessionSaved(t, fixture.SessionsDir, sessionID)
	})

	t.Run("resume with different workspace", func(t *testing.T) {
		// Create different workspace
		differentWorkspace := testutil.CreateTestWorkspace(t)

		// Try to resume with different workspace
		opts := session.SetupOptions{
			WorkspacePath: differentWorkspace,
			ResumeFromID:  sessionID,
			Slot:          1,
			SessionsDir:   fixture.SessionsDir,
		}

		result, err := session.Setup(opts)

		// Depending on implementation, this might:
		// 1. Fail with error (strict validation)
		// 2. Succeed with warning (allow workspace change)
		// 3. Succeed and use new workspace

		if err != nil {
			// If it fails, verify error is helpful
			testutil.AssertErrorContains(t, err, "workspace")
		} else {
			// If it succeeds, verify it works correctly
			testutil.AssertContainerRunning(t, result.Manager)

			// Cleanup
			cleanupOpts := session.CleanupOptions{
				ContainerName: result.ContainerName,
				SessionID:     sessionID,
				SessionsDir:   fixture.SessionsDir,
				SaveSession:   false,
			}
			_ = session.Cleanup(cleanupOpts)
		}
	})
}

// TestResumeNonExistentSession verifies error handling for invalid session ID
//
// Scenarios covered:
// - Resume non-existent session ID
// - Resume with corrupted session directory
// - Verify helpful error messages
func TestResumeNonExistentSession(t *testing.T) {
	testutil.SkipIfNoIncus(t)

	fixture := testutil.NewSessionFixture(t)

	t.Run("resume non-existent session", func(t *testing.T) {
		opts := session.SetupOptions{
			WorkspacePath: fixture.Workspace,
			ResumeFromID:  "non-existent-session-id",
			Slot:          1,
			SessionsDir:   fixture.SessionsDir,
		}

		_, err := session.Setup(opts)

		if err == nil {
			t.Error("Expected error when resuming non-existent session, but got none")
		}

		// Verify error is helpful
		testutil.AssertErrorContains(t, err, "session")
	})

	t.Run("resume with empty .claude directory", func(t *testing.T) {
		// Create session directory but with empty .claude
		sessionID, _ := session.GenerateSessionID()
		sessionDir := filepath.Join(fixture.SessionsDir, sessionID, ".claude")
		err := os.MkdirAll(sessionDir, 0755)
		testutil.AssertNoError(t, err)

		opts := session.SetupOptions{
			WorkspacePath: fixture.Workspace,
			ResumeFromID:  sessionID,
			Slot:          1,
			SessionsDir:   fixture.SessionsDir,
		}

		result, err := session.Setup(opts)

		// Empty .claude directory should either:
		// 1. Resume successfully with empty state
		// 2. Fail with helpful error

		if err != nil {
			// If it fails, verify error mentions .claude
			testutil.AssertErrorContains(t, err, "claude")
		} else {
			// If it succeeds, verify container is running
			testutil.AssertContainerRunning(t, result.Manager)

			// Cleanup
			cleanupOpts := session.CleanupOptions{
				ContainerName: result.ContainerName,
				SessionID:     sessionID,
				SessionsDir:   fixture.SessionsDir,
				SaveSession:   false,
			}
			_ = session.Cleanup(cleanupOpts)
		}
	})
}

// TestResumeMultipleTimes verifies session can be resumed multiple times
//
// Scenarios covered:
// - Resume session, make changes, save
// - Resume again, verify both old and new state
// - Verify state accumulates across resumes
func TestResumeMultipleTimes(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	var sessionID string

	t.Run("first session", func(t *testing.T) {
		result := fixture.Setup(t)
		sessionID = fixture.SessionID

		// Create first file
		claudeDir := filepath.Join(result.HomeDir, ".claude")
		file1 := filepath.Join(claudeDir, "file1.txt")
		err := result.Manager.CreateFile(file1, "content from first session")
		testutil.AssertNoError(t, err)

		// Save
		err = fixture.Cleanup(t, true)
		testutil.AssertNoError(t, err)
	})

	t.Run("second session (first resume)", func(t *testing.T) {
		opts := session.SetupOptions{
			WorkspacePath: fixture.Workspace,
			ResumeFromID:  sessionID,
			Slot:          1,
			SessionsDir:   fixture.SessionsDir,
		}

		result, err := session.Setup(opts)
		testutil.AssertNoError(t, err)

		// Verify first file exists
		claudeDir := filepath.Join(result.HomeDir, ".claude")
		file1 := filepath.Join(claudeDir, "file1.txt")
		testutil.AssertFileContent(t, result.Manager, file1, "content from first session")

		// Create second file
		file2 := filepath.Join(claudeDir, "file2.txt")
		err = result.Manager.CreateFile(file2, "content from second session")
		testutil.AssertNoError(t, err)

		// Save again
		cleanupOpts := session.CleanupOptions{
			ContainerName: result.ContainerName,
			SessionID:     sessionID,
			SessionsDir:   fixture.SessionsDir,
			SaveSession:   true,
		}
		err = session.Cleanup(cleanupOpts)
		testutil.AssertNoError(t, err)
	})

	t.Run("third session (second resume)", func(t *testing.T) {
		opts := session.SetupOptions{
			WorkspacePath: fixture.Workspace,
			ResumeFromID:  sessionID,
			Slot:          1,
			SessionsDir:   fixture.SessionsDir,
		}

		result, err := session.Setup(opts)
		testutil.AssertNoError(t, err)

		// Verify both files exist
		claudeDir := filepath.Join(result.HomeDir, ".claude")

		file1 := filepath.Join(claudeDir, "file1.txt")
		testutil.AssertFileContent(t, result.Manager, file1, "content from first session")

		file2 := filepath.Join(claudeDir, "file2.txt")
		testutil.AssertFileContent(t, result.Manager, file2, "content from second session")

		// Cleanup final
		cleanupOpts := session.CleanupOptions{
			ContainerName: result.ContainerName,
			SessionID:     sessionID,
			SessionsDir:   fixture.SessionsDir,
			SaveSession:   false,
		}
		err = session.Cleanup(cleanupOpts)
		testutil.AssertNoError(t, err)
	})
}
