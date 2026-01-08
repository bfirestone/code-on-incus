// +build integration,scenarios

package cli

import (
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestCleanCommandBasic verifies basic cleanup functionality
//
// Scenarios covered:
// - Create container
// - Run cleanup
// - Verify container removed
func TestCleanCommandBasic(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("cleanup removes container", func(t *testing.T) {
		result := fixture.Setup(t)

		// Verify running
		testutil.AssertContainerRunning(t, result.Manager)

		// Cleanup
		err := fixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)

		// Verify not running
		testutil.AssertContainerNotRunning(t, result.Manager)
	})
}

// TestCleanCommandWithSessions verifies session cleanup
//
// Scenarios covered:
// - Create and save session
// - Verify session saved
// - Clean sessions
func TestCleanCommandWithSessions(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("cleanup saves and removes session", func(t *testing.T) {
		_ = fixture.Setup(t)

		// Cleanup with save
		err := fixture.Cleanup(t, true)
		testutil.AssertNoError(t, err)

		// Verify session saved
		testutil.AssertSessionSaved(t, fixture.SessionsDir, fixture.SessionID)
	})
}

// TestCleanCommandForce verifies force cleanup
//
// Scenarios covered:
// - Force cleanup without confirmation
// - Cleanup running containers
func TestCleanCommandForce(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("force cleanup", func(t *testing.T) {
		result := fixture.Setup(t)

		// Force stop and delete
		err := result.Manager.Stop(true)
		testutil.AssertNoError(t, err)

		err = result.Manager.Delete(true)
		testutil.AssertNoError(t, err)

		// Verify removed
		testutil.AssertContainerNotRunning(t, result.Manager)
	})
}
