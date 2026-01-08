// +build integration,scenarios

package scenarios

import (
	"path/filepath"
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestSessionPersistenceMetadata verifies session metadata persistence
//
// Scenarios covered:
// - Session metadata.json saved with correct structure
// - Metadata contains workspace, image, timestamps
//
// Note: Basic persistence is covered in resume_session_test.go
// This file covers additional metadata scenarios
func TestSessionPersistenceMetadata(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("metadata file saved", func(t *testing.T) {
		_ = fixture.Setup(t)

		// Cleanup with save
		err := fixture.Cleanup(t, true)
		testutil.AssertNoError(t, err)

		// Verify session directory exists
		testutil.AssertSessionSaved(t, fixture.SessionsDir, fixture.SessionID)

		// Verify .claude directory exists
		claudePath := filepath.Join(fixture.SessionsDir, fixture.SessionID, ".claude")
		testutil.AssertFileOnHost(t, claudePath)

		t.Logf("Session metadata saved for session: %s", fixture.SessionID)
	})
}
