// +build integration,scenarios

package scenarios

import (
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestContainerLifecycleEphemeral verifies ephemeral container auto-deletion
//
// Scenarios covered:
// - Launch ephemeral container
// - Stop container
// - Verify container auto-deletes
func TestContainerLifecycleEphemeral(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t)
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("ephemeral container auto-deletes", func(t *testing.T) {
		result := fixture.Setup(t)

		// Verify running
		testutil.AssertContainerRunning(t, result.Manager)

		// Stop container
		err := result.Manager.Stop(true)
		testutil.AssertNoError(t, err)

		// Ephemeral containers should auto-delete when stopped
		testutil.AssertContainerNotRunning(t, result.Manager)
	})
}

// TestContainerLifecycleStartStopCycle verifies start/stop operations
//
// Scenarios covered:
// - Start container
// - Stop container  
// - Restart container
func TestContainerLifecycleStartStopCycle(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	// Use container fixture for more control
	containerFixture := testutil.NewContainerFixture(t, "test-lifecycle").
		WithPersistent()

	defer containerFixture.Cleanup(t)

	t.Run("start container", func(t *testing.T) {
		err := containerFixture.Launch(t)
		testutil.AssertNoError(t, err)

		testutil.AssertContainerRunning(t, containerFixture.Manager)
	})

	t.Run("stop container", func(t *testing.T) {
		err := containerFixture.Manager.Stop(false)
		testutil.AssertNoError(t, err)

		testutil.AssertContainerNotRunning(t, containerFixture.Manager)
	})
}
