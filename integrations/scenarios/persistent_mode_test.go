// +build integration,scenarios

package scenarios

import (
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/session"
	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestPersistentModeBasic verifies basic persistent mode functionality
//
// Scenarios covered:
// - First launch creates non-ephemeral container
// - Cleanup stops but doesn't delete container
// - Second launch reuses existing container
// - Container state persists across sessions
func TestPersistentModeBasic(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t).WithPersistent()
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	var firstResult *session.SetupResult

	t.Run("first launch creates persistent container", func(t *testing.T) {
		firstResult = fixture.Setup(t)

		testutil.AssertContainerRunning(t, firstResult.Manager)

		// Verify container is non-ephemeral
		output, err := container.IncusOutput("list", fixture.ContainerName, "--format=csv", "--columns=e")
		testutil.AssertNoError(t, err)
		if output == "true\n" {
			t.Error("Expected non-ephemeral container, got ephemeral")
		}
	})

	t.Run("create state in container", func(t *testing.T) {
		// Install a test package to verify persistence
		cmd := "touch /tmp/persistent-marker.txt && echo 'test data' > /tmp/persistent-marker.txt"
		testutil.AssertCommandSucceeds(t, firstResult.Manager, cmd)

		// Verify file exists
		testutil.AssertFileContent(t, firstResult.Manager, "/tmp/persistent-marker.txt", "test data\n")
	})

	t.Run("cleanup preserves container", func(t *testing.T) {
		err := fixture.Cleanup(t, true)
		testutil.AssertNoError(t, err)

		// Container should still exist (stopped)
		exists, err := firstResult.Manager.Exists()
		testutil.AssertNoError(t, err)
		if !exists {
			t.Error("Expected container to still exist after cleanup")
		}

		// Container should not be running
		running, err := firstResult.Manager.Running()
		testutil.AssertNoError(t, err)
		if running {
			t.Error("Expected container to be stopped after cleanup")
		}
	})

	t.Run("second launch reuses existing container", func(t *testing.T) {
		secondResult := fixture.Setup(t)

		// Should be same container name
		if secondResult.ContainerName != firstResult.ContainerName {
			t.Errorf("Expected same container name: got %s, want %s",
				secondResult.ContainerName, firstResult.ContainerName)
		}

		// Container should be running
		testutil.AssertContainerRunning(t, secondResult.Manager)

		// Previous state should be preserved
		testutil.AssertFileContent(t, secondResult.Manager, "/tmp/persistent-marker.txt", "test data\n")
	})
}

// TestPersistentModeToolInstallation verifies installed tools persist
//
// Scenarios covered:
// - Install package in first session
// - Package still available in second session
// - No reinstallation needed
func TestPersistentModeToolInstallation(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t).WithPersistent().WithPrivileged()
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	// Check if privileged image exists
	exists, err := container.ImageExists(session.PrivilegedImage)
	if err != nil || !exists {
		t.Skipf("Privileged image %s not available", session.PrivilegedImage)
	}

	var result *session.SetupResult

	t.Run("first session - install tool", func(t *testing.T) {
		result = fixture.Setup(t)

		// Install jq (lightweight JSON processor)
		cmd := "sudo apt-get update -qq && sudo apt-get install -y -qq jq > /dev/null 2>&1"
		_, err := result.Manager.ExecCommand(cmd, testutil.ExecOpts())
		if err != nil {
			t.Skipf("Could not install jq (apt-get might not be available): %v", err)
		}

		// Verify jq installed
		testutil.AssertCommandSucceeds(t, result.Manager, "which jq")
	})

	t.Run("cleanup and restart", func(t *testing.T) {
		err := fixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)

		// Second session
		result = fixture.Setup(t)
	})

	t.Run("second session - tool still available", func(t *testing.T) {
		// jq should still be installed (no reinstall needed)
		output := testutil.AssertCommandSucceeds(t, result.Manager, "which jq")
		if output != "/usr/bin/jq\n" {
			t.Errorf("Expected jq at /usr/bin/jq, got: %s", output)
		}

		// Verify jq works
		testCmd := "echo '{\"test\": \"value\"}' | jq .test"
		output = testutil.AssertCommandSucceeds(t, result.Manager, testCmd)
		if output != "\"value\"\n" {
			t.Errorf("Expected jq to work, got: %s", output)
		}
	})
}

// TestPersistentModeVsEphemeral verifies difference between modes
//
// Scenarios covered:
// - Ephemeral mode: container deleted on cleanup
// - Persistent mode: container kept on cleanup
func TestPersistentModeVsEphemeral(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	t.Run("ephemeral mode deletes container", func(t *testing.T) {
		fixture := testutil.NewSessionFixture(t) // NOT persistent
		defer testutil.CleanupTestContainers(t, fixture.ContainerName)

		result := fixture.Setup(t)
		testutil.AssertContainerRunning(t, result.Manager)

		// Cleanup
		err := fixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)

		// Container should NOT exist
		exists, err := result.Manager.Exists()
		testutil.AssertNoError(t, err)
		if exists {
			t.Error("Expected ephemeral container to be deleted after cleanup")
		}
	})

	t.Run("persistent mode keeps container", func(t *testing.T) {
		fixture := testutil.NewSessionFixture(t).WithPersistent()
		defer testutil.CleanupTestContainers(t, fixture.ContainerName)

		result := fixture.Setup(t)
		testutil.AssertContainerRunning(t, result.Manager)

		// Cleanup
		err := fixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)

		// Container SHOULD exist (stopped)
		exists, err := result.Manager.Exists()
		testutil.AssertNoError(t, err)
		if !exists {
			t.Error("Expected persistent container to still exist after cleanup")
		}

		running, err := result.Manager.Running()
		testutil.AssertNoError(t, err)
		if running {
			t.Error("Expected persistent container to be stopped after cleanup")
		}
	})
}

// TestPersistentModeAlreadyRunning verifies handling of already-running container
//
// Scenarios covered:
// - Container already running (not stopped after previous session)
// - Setup reuses running container without restart
func TestPersistentModeAlreadyRunning(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t).WithPersistent()
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("first launch", func(t *testing.T) {
		result := fixture.Setup(t)
		testutil.AssertContainerRunning(t, result.Manager)

		// Create marker file
		testutil.AssertCommandSucceeds(t, result.Manager, "echo 'first' > /tmp/marker.txt")
	})

	// DON'T cleanup - leave container running

	t.Run("second launch with already-running container", func(t *testing.T) {
		result := fixture.Setup(t)

		// Should reuse running container
		testutil.AssertContainerRunning(t, result.Manager)

		// Marker should still exist
		testutil.AssertFileContent(t, result.Manager, "/tmp/marker.txt", "first\n")
	})
}

// TestPersistentModeMultipleCycles verifies repeated start/stop cycles
//
// Scenarios covered:
// - Multiple cleanup/setup cycles
// - State persists across all cycles
// - No resource leaks
func TestPersistentModeMultipleCycles(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t).WithPersistent()
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	cycles := 3
	for i := 1; i <= cycles; i++ {
		t.Run(testutil.FormatTestName("cycle %d", i), func(t *testing.T) {
			result := fixture.Setup(t)
			testutil.AssertContainerRunning(t, result.Manager)

			// Add marker for this cycle
			cmd := testutil.FormatCommand("echo 'cycle-%d' >> /tmp/cycles.txt", i)
			testutil.AssertCommandSucceeds(t, result.Manager, cmd)

			// Cleanup
			err := fixture.Cleanup(t, false)
			testutil.AssertNoError(t, err)
		})
	}

	t.Run("verify all cycles recorded", func(t *testing.T) {
		result := fixture.Setup(t)

		// Should have all cycle markers
		output := testutil.AssertCommandSucceeds(t, result.Manager, "cat /tmp/cycles.txt")
		expected := "cycle-1\ncycle-2\ncycle-3\n"
		if output != expected {
			t.Errorf("Expected all cycles recorded:\ngot: %q\nwant: %q", output, expected)
		}
	})
}
