// +build integration,scenarios

package scenarios

import (
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestCleanupWithSave verifies cleanup saves session data
//
// Scenarios covered:
// - Normal cleanup with SaveSession=true
// - Verify session data saved
// - Verify container deleted
func TestCleanupWithSave(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("cleanup with save", func(t *testing.T) {
		result := fixture.Setup(t)

		// Create some .claude state
		claudeFile := result.HomeDir + "/.claude/test.txt"
		err := result.Manager.CreateFile(claudeFile, "test state")
		testutil.AssertNoError(t, err)

		// Cleanup with save
		err = fixture.Cleanup(t, true)
		testutil.AssertNoError(t, err)

		// Verify session saved
		testutil.AssertSessionSaved(t, fixture.SessionsDir, fixture.SessionID)

		// Verify container deleted
		testutil.AssertContainerNotRunning(t, result.Manager)
	})
}

// TestCleanupWithoutSave verifies cleanup without saving
//
// Scenarios covered:
// - Cleanup with SaveSession=false
// - Verify session data not saved
// - Verify container deleted
func TestCleanupWithoutSave(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("cleanup without save", func(t *testing.T) {
		result := fixture.Setup(t)

		// Cleanup without save
		err := fixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)

		// Verify session not saved
		testutil.AssertSessionNotSaved(t, fixture.SessionsDir, fixture.SessionID)

		// Verify container deleted
		testutil.AssertContainerNotRunning(t, result.Manager)
	})
}

// TestCleanupContainerAlreadyDeleted verifies cleanup handles missing container
//
// Scenarios covered:
// - Container already deleted before cleanup
// - Cleanup completes gracefully
func TestCleanupContainerAlreadyDeleted(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("cleanup after manual deletion", func(t *testing.T) {
		result := fixture.Setup(t)

		// Manually delete container
		_ = result.Manager.Stop(true)
		_ = result.Manager.Delete(true)

		// Cleanup should handle this gracefully
		err := fixture.Cleanup(t, false)

		// Should either succeed or fail gracefully
		if err != nil {
			t.Logf("Cleanup with missing container returned error: %v", err)
		}
	})
}
