// +build integration,scenarios

package scenarios

import (
	"strings"
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/session"
	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestPersistentContainerRealIncus verifies persistent containers with real Incus
//
// Scenarios covered:
// - Ephemeral container is deleted on cleanup
// - Persistent container is kept on cleanup
// - Persistent container can be restarted
// - Container state persists across restart
func TestPersistentContainerRealIncus(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	t.Run("ephemeral container deleted on cleanup", func(t *testing.T) {
		fixture := testutil.NewSessionFixture(t) // NOT persistent
		defer testutil.CleanupTestContainers(t, fixture.ContainerName)

		// Setup
		result := fixture.Setup(t)
		testutil.AssertContainerRunning(t, result.Manager)

		// Create marker
		testutil.AssertCommandSucceeds(t, result.Manager, "echo 'ephemeral' > /tmp/marker.txt")

		// Cleanup
		err := fixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)

		// Verify container deleted
		exists, err := result.Manager.Exists()
		testutil.AssertNoError(t, err)
		if exists {
			t.Error("Ephemeral container should be deleted after cleanup")
		}
	})

	t.Run("persistent container kept on cleanup", func(t *testing.T) {
		fixture := testutil.NewSessionFixture(t).WithPersistent()
		defer testutil.CleanupTestContainers(t, fixture.ContainerName)

		// Setup
		result := fixture.Setup(t)
		testutil.AssertContainerRunning(t, result.Manager)

		// Create marker
		testutil.AssertCommandSucceeds(t, result.Manager, "echo 'persistent' > /tmp/marker.txt")

		// Cleanup
		err := fixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)

		// Verify container still exists
		exists, err := result.Manager.Exists()
		testutil.AssertNoError(t, err)
		if !exists {
			t.Fatal("Persistent container should exist after cleanup")
		}

		// Verify it's stopped
		running, err := result.Manager.Running()
		testutil.AssertNoError(t, err)
		if running {
			t.Error("Persistent container should be stopped after cleanup")
		}
	})

	t.Run("persistent container restart preserves state", func(t *testing.T) {
		fixture := testutil.NewSessionFixture(t).WithPersistent()
		defer testutil.CleanupTestContainers(t, fixture.ContainerName)

		// First session
		t.Log("First session - create state")
		result1 := fixture.Setup(t)
		testutil.AssertContainerRunning(t, result1.Manager)

		// Create marker file using echo command
		markerContent := "persistent state test"
		cmd := "echo '" + markerContent + "' > /tmp/persistent-marker.txt && ls -la /tmp/persistent-marker.txt"
		output, err := result1.Manager.ExecCommand(cmd, testutil.ExecOpts())
		t.Logf("Echo command output: %q, error: %v", output, err)
		testutil.AssertNoError(t, err)

		// Verify file exists
		testutil.AssertFileExists(t, result1.Manager, "/tmp/persistent-marker.txt")

		// Read and verify content
		readCmd := "cat /tmp/persistent-marker.txt"
		content, err := result1.Manager.ExecCommand(readCmd, testutil.ExecOpts())
		testutil.AssertNoError(t, err)
		t.Logf("File content after creation: %q", content)

		if content != markerContent+"\n" {
			t.Errorf("File content mismatch: got %q, want %q", content, markerContent+"\n")
		}

		// Cleanup (stop but keep)
		err = fixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)

		// Second session
		t.Log("Second session - verify state preserved")
		result2 := fixture.Setup(t)

		// Should be same container
		if result2.ContainerName != result1.ContainerName {
			t.Errorf("Expected same container name: got %s, want %s",
				result2.ContainerName, result1.ContainerName)
		}

		// Container should be running again
		testutil.AssertContainerRunning(t, result2.Manager)

		// Marker file should still exist with correct content
		content2, err := result2.Manager.ExecCommand("cat /tmp/persistent-marker.txt", testutil.ExecOpts())
		testutil.AssertNoError(t, err)
		t.Logf("File content after restart: %q", content2)

		if content2 != markerContent+"\n" {
			t.Errorf("File content not preserved across restart: got %q, want %q", content2, markerContent+"\n")
		}
	})
}

// TestPersistentToolInstallation verifies tools persist across sessions
//
// This is the key use case: install packages once, use them forever
func TestPersistentToolInstallation(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	// Need privileged for apt-get
	fixture := testutil.NewSessionFixture(t).WithPersistent().WithPrivileged()
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	// Check if privileged image exists
	exists, err := container.ImageExists(session.PrivilegedImage)
	if err != nil || !exists {
		t.Skipf("Privileged image %s not available", session.PrivilegedImage)
	}

	t.Run("install tool in first session", func(t *testing.T) {
		result := fixture.Setup(t)

		// Check if jq already installed (shouldn't be in fresh container)
		_, err := result.Manager.ExecCommand("which jq", testutil.ExecOpts())
		if err == nil {
			t.Log("Warning: jq already installed (container might not be fresh)")
		}

		// Install jq (lightweight JSON processor)
		t.Log("Installing jq...")
		cmd := "sudo apt-get update -qq && sudo apt-get install -y -qq jq > /dev/null 2>&1"
		_, err = result.Manager.ExecCommand(cmd, testutil.ExecOpts())
		if err != nil {
			t.Skipf("Could not install jq: %v", err)
		}

		// Verify jq installed
		output := testutil.AssertCommandSucceeds(t, result.Manager, "which jq")
		jqPath := strings.TrimSpace(output)
		if jqPath != "/usr/bin/jq" {
			t.Logf("jq installed at: %s (expected /usr/bin/jq)", jqPath)
		}

		// Verify jq works
		testCmd := "echo '{\"test\": \"value\"}' | jq -r .test"
		output = testutil.AssertCommandSucceeds(t, result.Manager, testCmd)
		if strings.TrimSpace(output) != "value" {
			t.Errorf("jq test failed: got %q, want %q", output, "value")
		}

		// Cleanup
		err = fixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)
	})

	t.Run("tool available in second session without reinstall", func(t *testing.T) {
		result := fixture.Setup(t)

		// jq should still be available (NO reinstall needed)
		output := testutil.AssertCommandSucceeds(t, result.Manager, "which jq")
		jqPath := strings.TrimSpace(output)
		if !strings.Contains(jqPath, "jq") {
			t.Errorf("jq not found in second session: %s", jqPath)
		}

		// Verify it still works
		testCmd := "echo '{\"persistent\": \"true\"}' | jq -r .persistent"
		output = testutil.AssertCommandSucceeds(t, result.Manager, testCmd)
		if strings.TrimSpace(output) != "true" {
			t.Errorf("jq test failed in second session: got %q", output)
		}

		t.Log("✅ Tool persisted across sessions - no reinstall needed!")
	})
}

// TestPersistentBuildArtifacts verifies build artifacts persist
func TestPersistentBuildArtifacts(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t).WithPersistent()
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	var buildOutput string

	t.Run("create build artifacts", func(t *testing.T) {
		result := fixture.Setup(t)

		// Create a mock build directory with artifacts
		cmd := `
mkdir -p /tmp/build/dist
echo "compiled binary" > /tmp/build/dist/app
echo "build-123" > /tmp/build/dist/version.txt
echo "cache data" > /tmp/build/.cache
ls -la /tmp/build/dist
`
		output := testutil.AssertCommandSucceeds(t, result.Manager, cmd)
		buildOutput = output
		t.Logf("Build artifacts created:\n%s", buildOutput)

		// Cleanup
		err := fixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)
	})

	t.Run("build artifacts persist across restart", func(t *testing.T) {
		result := fixture.Setup(t)

		// Verify artifacts still exist
		testutil.AssertFileContent(t, result.Manager, "/tmp/build/dist/app", "compiled binary\n")
		testutil.AssertFileContent(t, result.Manager, "/tmp/build/dist/version.txt", "build-123\n")
		testutil.AssertFileContent(t, result.Manager, "/tmp/build/.cache", "cache data\n")

		t.Log("✅ Build artifacts persisted - no rebuild needed!")
	})
}

// TestPersistentMultipleCycles verifies repeated stop/start cycles
func TestPersistentMultipleCycles(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	fixture := testutil.NewSessionFixture(t).WithPersistent()
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	cycles := 3

	for i := 1; i <= cycles; i++ {
		t.Run(testutil.FormatTestName("cycle %d", i), func(t *testing.T) {
			result := fixture.Setup(t)

			// Append to file
			cmd := testutil.FormatCommand("echo 'cycle-%d' >> /tmp/cycles.log", i)
			testutil.AssertCommandSucceeds(t, result.Manager, cmd)

			// Cleanup
			err := fixture.Cleanup(t, false)
			testutil.AssertNoError(t, err)
		})
	}

	t.Run("verify all cycles recorded", func(t *testing.T) {
		result := fixture.Setup(t)

		// Should have all cycle entries
		output := testutil.AssertCommandSucceeds(t, result.Manager, "cat /tmp/cycles.log")

		for i := 1; i <= cycles; i++ {
			expected := testutil.FormatCommand("cycle-%d", i)
			if !strings.Contains(output, expected) {
				t.Errorf("Missing cycle %d in output:\n%s", i, output)
			}
		}

		// Count lines (should be exactly 3)
		lineCount := strings.Count(strings.TrimSpace(output), "\n") + 1
		if lineCount != cycles {
			t.Errorf("Expected %d cycle entries, got %d:\n%s", cycles, lineCount, output)
		}

		t.Logf("✅ All %d cycles recorded correctly", cycles)
	})
}

// TestPersistentVsEphemeralSideBySide compares both modes
func TestPersistentVsEphemeralSideBySide(t *testing.T) {
	testutil.SkipIfNoIncus(t)
	testutil.EnsureTestImage(t)

	ephemeralFixture := testutil.NewSessionFixture(t).WithSlot(91) // NOT persistent
	persistentFixture := testutil.NewSessionFixture(t).WithSlot(92).WithPersistent()

	defer testutil.CleanupTestContainers(t, ephemeralFixture.ContainerName)
	defer testutil.CleanupTestContainers(t, persistentFixture.ContainerName)

	// Set same workspace for both
	persistentFixture.Workspace = ephemeralFixture.Workspace

	t.Run("setup both containers", func(t *testing.T) {
		ephResult := ephemeralFixture.Setup(t)
		persResult := persistentFixture.Setup(t)

		// Both should be running
		testutil.AssertContainerRunning(t, ephResult.Manager)
		testutil.AssertContainerRunning(t, persResult.Manager)

		// Different container names (different slots)
		if ephResult.ContainerName == persResult.ContainerName {
			t.Error("Should have different container names (different slots)")
		}
	})

	t.Run("cleanup and verify behavior difference", func(t *testing.T) {
		// Cleanup both
		err := ephemeralFixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)

		err = persistentFixture.Cleanup(t, false)
		testutil.AssertNoError(t, err)

		// Ephemeral should be gone
		ephMgr := container.NewManager(ephemeralFixture.ContainerName)
		exists, _ := ephMgr.Exists()
		if exists {
			t.Error("Ephemeral container should be deleted")
		}

		// Persistent should still exist (stopped)
		persMgr := container.NewManager(persistentFixture.ContainerName)
		exists, _ = persMgr.Exists()
		if !exists {
			t.Error("Persistent container should still exist")
		}

		running, _ := persMgr.Running()
		if running {
			t.Error("Persistent container should be stopped")
		}

		t.Log("✅ Ephemeral deleted, Persistent kept")
	})
}
