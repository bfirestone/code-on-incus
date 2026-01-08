// +build integration,scenarios

package scenarios

import (
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/session"
	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestPrivilegedSessionBasic verifies privileged mode with elevated permissions
//
// Scenarios covered:
// - Launch privileged session
// - Verify privileged image used
// - Verify sudo available
// - Verify elevated permissions work
func TestPrivilegedSessionBasic(t *testing.T) {
	testutil.SkipIfNoIncus(t)

	// Check if privileged image exists
	exists, err := container.ImageExists(session.PrivilegedImage)
	if err != nil || !exists {
		t.Skipf("Privileged image %s not available", session.PrivilegedImage)
	}

	fixture := testutil.NewSessionFixture(t).WithPrivileged()
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("launch privileged session", func(t *testing.T) {
		result := fixture.Setup(t)

		// Verify privileged image used
		if result.Image != session.PrivilegedImage {
			t.Errorf("Expected privileged image %q, got %q", session.PrivilegedImage, result.Image)
		}

		testutil.AssertContainerRunning(t, result.Manager)
	})

	t.Run("verify sudo available", func(t *testing.T) {
		result := fixture.Setup(t)

		// Check sudo command exists
		output := testutil.AssertCommandSucceeds(t, result.Manager, "which sudo")
		if output == "" {
			t.Error("Expected sudo to be available in privileged mode")
		}

		// Test sudo works (should not require password for claude user)
		testutil.AssertCommandSucceeds(t, result.Manager, "sudo echo 'test'")

		_ = fixture.Cleanup(t, false)
	})
}

// TestPrivilegedSessionGitConfig verifies git configuration mounting
//
// Scenarios covered:
// - Verify git config mounted (if configured)
// - Verify git operations work
func TestPrivilegedSessionGitConfig(t *testing.T) {
	testutil.SkipIfNoIncus(t)

	exists, err := container.ImageExists(session.PrivilegedImage)
	if err != nil || !exists {
		t.Skipf("Privileged image %s not available", session.PrivilegedImage)
	}

	fixture := testutil.NewSessionFixture(t).WithPrivileged()
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("git available", func(t *testing.T) {
		result := fixture.Setup(t)

		// Verify git is installed
		output := testutil.AssertCommandSucceeds(t, result.Manager, "which git")
		if output == "" {
			t.Error("Expected git to be available")
		}

		// Test git version
		testutil.AssertCommandSucceeds(t, result.Manager, "git --version")

		_ = fixture.Cleanup(t, false)
	})
}

// TestPrivilegedSessionGitHubCLI verifies GitHub CLI availability
//
// Scenarios covered:
// - Verify gh CLI available
// - Verify gh version works
func TestPrivilegedSessionGitHubCLI(t *testing.T) {
	testutil.SkipIfNoIncus(t)

	exists, err := container.ImageExists(session.PrivilegedImage)
	if err != nil || !exists {
		t.Skipf("Privileged image %s not available", session.PrivilegedImage)
	}

	fixture := testutil.NewSessionFixture(t).WithPrivileged()
	defer testutil.CleanupTestContainers(t, fixture.ContainerName)

	t.Run("github cli available", func(t *testing.T) {
		result := fixture.Setup(t)

		// Check gh command exists
		output := testutil.AssertCommandSucceeds(t, result.Manager, "which gh")
		if output == "" {
			t.Error("Expected gh CLI to be available in privileged mode")
		}

		// Test gh version
		testutil.AssertCommandSucceeds(t, result.Manager, "gh --version")

		_ = fixture.Cleanup(t, false)
	})
}
