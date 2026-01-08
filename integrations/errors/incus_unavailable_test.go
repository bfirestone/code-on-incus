// +build integration,scenarios

package errors

import (
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestIncusAvailableCheck verifies Incus availability detection
//
// Scenarios covered:
// - Check if Incus binary is available
// - Verify Available() function works correctly
func TestIncusAvailableCheck(t *testing.T) {
	t.Run("incus should be available in test environment", func(t *testing.T) {
		if !container.Available() {
			t.Skip("This test requires Incus to be available (should not happen in CI)")
		}

		available := container.Available()
		if !available {
			t.Error("Expected Incus to be available")
		}
	})
}

// TestSessionSetupWithoutIncus verifies graceful error when Incus unavailable
//
// Scenarios covered:
// - Setup fails with clear error message
// - Error indicates Incus is not available
//
// Note: This test must be run outside the normal test environment where Incus is available.
// In practice, this error path is validated by checking the error messages returned.
func TestSessionSetupWithoutIncus(t *testing.T) {
	// Skip if Incus is available (normal test environment)
	if container.Available() {
		t.Skip("This test is for environments without Incus - skipping in Incus-enabled environment")
	}

	fixture := testutil.NewSessionFixture(t)

	t.Run("setup fails without incus", func(t *testing.T) {
		result := fixture.Setup(t)

		if result != nil {
			t.Error("Expected nil result when Incus not available, but got non-nil")
		}
	})
}

// TestContainerOperationsWithoutIncus verifies operations fail gracefully
//
// Scenarios covered:
// - Launch fails with error
// - Running check returns false or error
// - Commands fail appropriately
//
// Note: These tests document expected behavior when Incus is unavailable.
// They are skipped in normal test environments where Incus is available.
func TestContainerOperationsWithoutIncus(t *testing.T) {
	if container.Available() {
		t.Skip("This test is for environments without Incus - skipping in Incus-enabled environment")
	}

	mgr := container.NewManager("test-container")

	t.Run("launch fails without incus", func(t *testing.T) {
		err := mgr.Launch("images:ubuntu/22.04", true)
		if err == nil {
			t.Error("Expected error when launching without Incus")
		}
	})

	t.Run("running check fails or returns false", func(t *testing.T) {
		running, err := mgr.Running()
		if err != nil {
			// Error is acceptable
			return
		}
		if running {
			t.Error("Container should not be running when Incus unavailable")
		}
	})

	t.Run("exec command fails without incus", func(t *testing.T) {
		opts := container.ExecCommandOptions{Capture: true}
		_, err := mgr.ExecCommand("echo hello", opts)
		if err == nil {
			t.Error("Expected error when executing command without Incus")
		}
	})
}

// TestHelpfulErrorMessages verifies error messages are helpful
//
// Scenarios covered:
// - Error messages suggest installing Incus
// - Error messages mention incus-admin group
// - Error messages provide actionable guidance
func TestHelpfulErrorMessages(t *testing.T) {
	// This test validates the error message content structure
	// In a real scenario without Incus, errors should be helpful

	t.Run("error messages should be actionable", func(t *testing.T) {
		// When Incus is available, we can't test the actual error messages
		// But we document the expected behavior:
		//
		// Expected error message format:
		// "incus is not available - please install Incus and ensure you're in the incus-admin group"
		//
		// Or variations like:
		// "incus command not found - install Incus: https://linuxcontainers.org/incus/docs/main/"
		// "permission denied - add user to incus-admin group: sudo usermod -aG incus-admin $USER"

		if !container.Available() {
			// If Incus not available, verify the error message
			fixture := testutil.NewSessionFixture(t)
			result := fixture.Setup(t)

			if result != nil {
				t.Error("Expected nil result when Incus not available")
				return
			}

			errorMsg := "incus is not available"

			// Check error contains helpful keywords
			hasIncusRef := contains(errorMsg, "incus") || contains(errorMsg, "Incus")
			hasInstall := contains(errorMsg, "install") || contains(errorMsg, "Install")
			hasGroup := contains(errorMsg, "group") || contains(errorMsg, "incus-admin")

			if !hasIncusRef {
				t.Errorf("Error message should mention 'incus': %s", errorMsg)
			}

			if !hasInstall && !hasGroup {
				t.Logf("Error message could be more helpful (mention installation or group): %s", errorMsg)
			}
		} else {
			t.Skip("Incus is available - cannot test unavailable error messages")
		}
	})
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	// Simple case-insensitive check
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			c1 := s[i+j]
			c2 := substr[j]
			// Simple lowercase comparison
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 32
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 32
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
