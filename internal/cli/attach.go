package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/spf13/cobra"
)

var (
	attachWithBash bool
)

var attachCmd = &cobra.Command{
	Use:   "attach [container-name]",
	Short: "Attach to a running Claude session",
	Long: `Attach to a running Claude session in a container.

If no container name is provided, lists all running sessions.
If only one session is running, attaches to it automatically.

Examples:
  coi attach                    # List sessions or auto-attach if only one
  coi attach claude-abc123-1    # Attach to specific session
  coi attach --bash             # Attach to bash shell instead of tmux session
  coi attach coi-123 --bash     # Attach to specific container with bash`,
	RunE: attachCommand,
}

func init() {
	attachCmd.Flags().BoolVar(&attachWithBash, "bash", false, "Attach to bash shell instead of tmux session")
	rootCmd.AddCommand(attachCmd)
}

func attachCommand(cmd *cobra.Command, args []string) error {
	// List all running containers
	containers, err := container.ListContainers("claude-.*")
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("No active Claude sessions")
		return nil
	}

	var targetContainer string

	// If container name provided, use it
	if len(args) > 0 {
		targetContainer = args[0]
		// Verify it exists and is running
		found := false
		for _, c := range containers {
			if c == targetContainer {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("container %s not found or not running", targetContainer)
		}
	} else if len(containers) == 1 {
		// Auto-attach if only one session
		targetContainer = containers[0]
		fmt.Printf("Attaching to %s...\n", targetContainer)
	} else {
		// Multiple sessions - show list
		fmt.Println("Active Claude sessions:")
		for i, c := range containers {
			mgr := container.NewManager(c)
			running, err := mgr.Running()
			if err != nil || !running {
				continue
			}
			fmt.Printf("  %d. %s\n", i+1, c)
		}
		fmt.Printf("\nUse: coi attach <container-name>\n")
		return nil
	}

	// Attach to container (tmux or bash)
	if attachWithBash {
		return attachToContainerWithBash(targetContainer)
	}
	return attachToContainer(targetContainer)
}

func attachToContainer(containerName string) error {
	// Build the command to attach as claude user
	// Use tmux attach which will auto-find the session
	tmuxCmd := "tmux attach"

	// Execute with incus exec, running as claude user
	args := []string{
		"exec",
		containerName,
		"--",
		"su", "-", "claude",
		"-c", tmuxCmd,
	}

	// Use incus command
	incusCmd := exec.Command("incus", args...)
	incusCmd.Stdin = os.Stdin
	incusCmd.Stdout = os.Stdout
	incusCmd.Stderr = os.Stderr

	err := incusCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to attach to session: %w", err)
	}

	return nil
}

func attachToContainerWithBash(containerName string) error {
	// Execute bash as claude user
	args := []string{
		"exec",
		containerName,
		"--",
		"su", "-", "claude",
		"-c", "cd /workspace && exec bash",
	}

	// Use incus command
	incusCmd := exec.Command("incus", args...)
	incusCmd.Stdin = os.Stdin
	incusCmd.Stdout = os.Stdout
	incusCmd.Stderr = os.Stderr

	err := incusCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to attach to container: %w", err)
	}

	return nil
}
