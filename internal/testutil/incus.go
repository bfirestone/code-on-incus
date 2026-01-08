package testutil

import (
	"fmt"
	"os"
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/container"
)

// SkipIfNoIncus skips the test if Incus is not available
func SkipIfNoIncus(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !container.Available() {
		t.Skip("Incus is not available - install Incus to run integration tests")
	}
}

// EnsureTestImage ensures the coi-sandbox test image is available
func EnsureTestImage(t *testing.T) {
	t.Helper()

	exists, err := container.ImageExists("coi-sandbox")
	if err != nil {
		t.Fatalf("Failed to check for test image: %v", err)
	}

	if !exists {
		t.Skipf("Test image coi-sandbox not available. Run: coi build sandbox")
	}
}

// CleanupTestContainers removes all test containers with given prefix
func CleanupTestContainers(t *testing.T, prefix string) {
	t.Helper()

	// List all containers with prefix
	output, err := container.IncusOutput("list", fmt.Sprintf("^%s", prefix), "--format=csv", "--columns=n")
	if err != nil {
		t.Logf("Warning: Failed to list test containers: %v", err)
		return
	}

	if output == "" {
		return
	}

	// Delete each container
	lines := splitLines(output)
	for _, name := range lines {
		if name == "" {
			continue
		}

		t.Logf("Cleaning up test container: %s", name)
		mgr := container.NewManager(name)
		_ = mgr.Stop(true)
		_ = mgr.Delete(true)
	}
}

// CreateTestWorkspace creates a temporary workspace directory
func CreateTestWorkspace(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Create a simple test file
	testFile := tmpDir + "/test.txt"
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	return tmpDir
}

// splitLines splits output by newlines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		line := s[start:]
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
