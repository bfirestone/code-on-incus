package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/spf13/cobra"
)

var (
	showAll bool
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "List available Incus images",
	Long: `List available Incus images for use with --image flag.

Shows both built COI images and available remote images.

Examples:
  coi images              # List COI images only
  coi images --all        # List all local images
`,
	RunE: imagesCommand,
}

func init() {
	imagesCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all local images, not just COI images")
}

func imagesCommand(cmd *cobra.Command, args []string) error {
	// Check if Incus is available
	if !container.Available() {
		return fmt.Errorf("incus is not available - please install Incus and ensure you're in the incus-admin group")
	}

	fmt.Println("Available Images:")
	fmt.Println()

	// Check COI images
	coiImages := []struct {
		alias       string
		description string
		buildCmd    string
	}{
		{"coi-sandbox", "Standard sandbox image (Claude CLI, Node.js, Docker, tmux)", "coi build sandbox"},
		{"coi-privileged", "Privileged image with Git/SSH/GitHub CLI", "coi build privileged"},
	}

	fmt.Println("COI Images:")
	for _, img := range coiImages {
		exists, err := container.ImageExists(img.alias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s - error checking: %v\n", img.alias, err)
			continue
		}

		if exists {
			fmt.Printf("  ✓ %s\n", img.alias)
			fmt.Printf("    %s\n", img.description)
		} else {
			fmt.Printf("  ✗ %s (not built)\n", img.alias)
			fmt.Printf("    %s\n", img.description)
			fmt.Printf("    Build with: %s\n", img.buildCmd)
		}
		fmt.Println()
	}

	if showAll {
		fmt.Println("All Local Images:")
		if err := listAllImages(); err != nil {
			return err
		}
	} else {
		fmt.Println("Tip: Use --all to see all local images")
	}

	fmt.Println()
	fmt.Println("Remote Images:")
	fmt.Println("  You can use any image from images.linuxcontainers.org:")
	fmt.Println("  - ubuntu:22.04, ubuntu:24.04")
	fmt.Println("  - debian:12, debian:11")
	fmt.Println("  - alpine:3.19")
	fmt.Println()
	fmt.Println("  Example: coi shell --image ubuntu:24.04")
	fmt.Println()
	fmt.Println("Custom Images:")
	fmt.Println("  Build your own: coi build custom --script setup.sh --name my-image")
	fmt.Println()

	return nil
}

// listAllImages lists all local Incus images
func listAllImages() error {
	mgr := container.NewManager("temp")

	// Get list of images using incus image list with sg wrapper
	output, err := mgr.ExecHostCommand("sg incus-admin -c 'incus image list --format=csv -c l,s,u'", true)
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println("  No local images found")
		return nil
	}

	fmt.Printf("  %-30s %-15s %s\n", "ALIAS", "SIZE", "UPLOAD DATE")
	fmt.Println("  " + strings.Repeat("-", 70))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		alias := parts[0]
		size := parts[1]
		uploadDate := parts[2]

		// Format size (convert bytes to human readable)
		sizeFormatted := formatSize(size)

		fmt.Printf("  %-30s %-15s %s\n", alias, sizeFormatted, uploadDate)
	}

	return nil
}

// formatSize converts byte string to human readable
func formatSize(sizeStr string) string {
	// Size is in bytes as string, convert to MB/GB
	var bytes int64
	fmt.Sscanf(sizeStr, "%d", &bytes)

	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1fGB", float64(bytes)/(1024*1024*1024))
}
