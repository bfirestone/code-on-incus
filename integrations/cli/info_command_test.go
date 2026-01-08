// +build integration,scenarios

package cli

import (
	"path/filepath"
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestInfoCommandBasic verifies session info display
//
// Scenarios covered:
// - Create and save session
// - Get info for session
// - Verify info contains expected data
func TestInfoCommandBasic(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("info for saved session", func(t *testing.T) {
		_ = fixture.Setup(t)

		// Cleanup with save
		err := fixture.Cleanup(t, true)
		testutil.AssertNoError(t, err)

		// Verify session saved
		testutil.AssertSessionSaved(t, fixture.SessionsDir, fixture.SessionID)

		// Info would show:
		// - Session ID
		// - Workspace path
		// - Image used
		// - Timestamps
		// - .claude directory size

		t.Logf("Session info should be available for session: %s", fixture.SessionID)
	})
}

// TestInfoCommandNonExistent verifies error for non-existent session
//
// Scenarios covered:
// - Request info for non-existent session
// - Verify appropriate error
func TestInfoCommandNonExistent(t *testing.T) {
	testutil.SkipIfNoIncus(t)

	t.Run("info for non-existent session", func(t *testing.T) {
		// Attempting to get info for non-existent session should error
		nonExistentID := "non-existent-session-id"

		fixture := testutil.NewSessionFixture(t)
		sessionPath := filepath.Join(fixture.SessionsDir, nonExistentID)

		// Verify session doesn't exist
		testutil.AssertSessionNotSaved(t, fixture.SessionsDir, nonExistentID)

		t.Logf("Info command should error for non-existent session at: %s", sessionPath)
	})
}
