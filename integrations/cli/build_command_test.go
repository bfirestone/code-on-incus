// +build integration,scenarios

package cli

import (
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/testutil"
)

// TestBuildCommand is a placeholder for image building scenarios
// Image building is tested in detail in integrations/images/ directory
//
// Scenarios covered in images/:
// - Build sandbox image
// - Build privileged image
// - Build custom image with script
// - Image versioning
// - Image rebuild
func TestBuildCommandPlaceholder(t *testing.T) {
	t.Skip("Image building tested in integrations/images/ directory")
}

// TestBuildCommandImageExists verifies checking if image exists
//
// Scenarios covered:
// - Check sandbox image exists (or skip test)
func TestBuildCommandImageExists(t *testing.T) {
	testutil.SkipIfNoIncus(t)

	t.Run("check base image exists", func(t *testing.T) {
		// Check if base ubuntu image exists
		exists, err := container.ImageExists("images:ubuntu/22.04")

		if err != nil {
			t.Skipf("Cannot check image existence: %v", err)
		}

		if !exists {
			t.Skipf("Base image images:ubuntu/22.04 not available")
		}
	})
}
