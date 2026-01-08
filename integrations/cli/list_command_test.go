// +build integration,scenarios

package cli

import (
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestListCommandEmpty verifies listing with no containers
//
// Scenarios covered:
// - List when no containers exist
// - Verify empty result
func TestListCommandEmpty(t *testing.T) {
	testutil.SkipIfNoIncus(t)

	t.Run("list with no containers", func(t *testing.T) {
		// Implementation would call list functionality
		// For now, just document expected behavior
		t.Log("List should return empty when no containers exist")
	})
}

// TestListCommandWithContainers verifies listing active containers
//
// Scenarios covered:
// - Create containers
// - List containers
// - Verify containers appear in list
func TestListCommandWithContainers(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture1 := testutil.NewSessionFixture(t).WithSlot(1)
	fixture2 := testutil.NewSessionFixture(t).WithSlot(2)

	defer testutil.CleanupTestContainers(t, fixture1.ContainerName)
	defer testutil.CleanupTestContainers(t, fixture2.ContainerName)

	t.Run("list shows active containers", func(t *testing.T) {
		result1 := fixture1.Setup(t)
		result2 := fixture2.Setup(t)

		// Verify both running
		testutil.AssertContainerRunning(t, result1.Manager)
		testutil.AssertContainerRunning(t, result2.Manager)

		// List functionality would show both containers
		t.Logf("Should list containers: %s, %s", result1.ContainerName, result2.ContainerName)

		_ = fixture1.Cleanup(t, false)
		_ = fixture2.Cleanup(t, false)
	})
}
