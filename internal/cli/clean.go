package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mensfeld/code-on-incus/internal/cleanup"
	"github.com/mensfeld/code-on-incus/internal/config"
	"github.com/mensfeld/code-on-incus/internal/container"
	"github.com/mensfeld/code-on-incus/internal/session"
	"github.com/spf13/cobra"
)

var (
	cleanAll      bool
	cleanForce    bool
	cleanSessions bool
	cleanOrphans  bool
	cleanDryRun   bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Cleanup containers, sessions, and orphaned resources",
	Long: `Cleanup stopped containers, old session data, and orphaned system resources.

By default, cleans only stopped containers. Use flags to control what gets cleaned.

Orphaned resources include:
- Orphaned veth interfaces (network pairs with no master bridge)
- Orphaned firewall rules (rules for container IPs that no longer exist)

Examples:
  coi clean                    # Clean stopped containers
  coi clean --sessions         # Clean saved session data
  coi clean --orphans          # Clean orphaned veths and firewall rules
  coi clean --all              # Clean everything
  coi clean --all --force      # Clean without confirmation
  coi clean --orphans --dry-run # Show what orphans would be cleaned
`,
	RunE: cleanCommand,
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Clean all containers, sessions, and orphaned resources")
	cleanCmd.Flags().BoolVar(&cleanForce, "force", false, "Skip confirmation prompts")
	cleanCmd.Flags().BoolVar(&cleanSessions, "sessions", false, "Clean saved session data")
	cleanCmd.Flags().BoolVar(&cleanOrphans, "orphans", false, "Clean orphaned veths and firewall rules")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be cleaned without making changes")
}

func cleanCommand(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get configured tool to determine tool-specific sessions directory
	toolInstance, err := getConfiguredTool(cfg)
	if err != nil {
		return err
	}

	// Get tool-specific sessions directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".coi")
	sessionsDir := session.GetSessionsDir(baseDir, toolInstance)

	cleaned := 0

	// Clean stopped containers
	if cleanAll || (!cleanSessions) {
		fmt.Println("Checking for stopped claude-on-incus containers...")

		containers, err := listActiveContainers()
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		stoppedContainers := []string{}
		for _, c := range containers {
			if c.Status == "Stopped" || c.Status == "STOPPED" {
				stoppedContainers = append(stoppedContainers, c.Name)
			}
		}

		if len(stoppedContainers) > 0 {
			fmt.Printf("Found %d stopped container(s):\n", len(stoppedContainers))
			for _, name := range stoppedContainers {
				fmt.Printf("  - %s\n", name)
			}

			if !cleanDryRun {
				if !cleanForce {
					fmt.Print("\nDelete these containers? [y/N]: ")
					var response string
					_, _ = fmt.Scanln(&response) // Ignore error, default to "no" if read fails
					if response != "y" && response != "Y" {
						fmt.Println("Cancelled.")
						return nil
					}
				}

				for _, name := range stoppedContainers {
					fmt.Printf("Deleting container %s...\n", name)
					mgr := container.NewManager(name)
					if err := mgr.Delete(true); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Failed to delete %s: %v\n", name, err)
					} else {
						cleaned++
					}
				}
			}
		} else {
			fmt.Println("  (no stopped containers found)")
		}
	}

	// Clean saved sessions
	if cleanAll || cleanSessions {
		fmt.Println("\nChecking for saved session data...")

		entries, err := os.ReadDir(sessionsDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read sessions directory: %w", err)
		}

		sessionDirs := []string{}
		for _, entry := range entries {
			if entry.IsDir() {
				sessionDirs = append(sessionDirs, entry.Name())
			}
		}

		if len(sessionDirs) > 0 {
			fmt.Printf("Found %d session(s):\n", len(sessionDirs))
			for _, name := range sessionDirs {
				fmt.Printf("  - %s\n", name)
			}

			if !cleanDryRun {
				if !cleanForce {
					fmt.Print("\nDelete all session data? [y/N]: ")
					var response string
					_, _ = fmt.Scanln(&response) // Ignore error, default to "no" if read fails
					if response != "y" && response != "Y" {
						fmt.Println("Cancelled.")
						return nil
					}
				}

				for _, name := range sessionDirs {
					sessionPath := filepath.Join(sessionsDir, name)
					fmt.Printf("Deleting session %s...\n", name)
					if err := os.RemoveAll(sessionPath); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Failed to delete %s: %v\n", name, err)
					} else {
						cleaned++
					}
				}
			}
		} else {
			fmt.Println("  (no saved sessions found)")
		}
	}

	// Clean orphaned resources (veths and firewall rules)
	if cleanAll || cleanOrphans {
		fmt.Println("\nScanning for orphaned resources...")

		orphans, err := cleanup.DetectAll()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to detect orphans: %v\n", err)
		} else {
			totalOrphans := len(orphans.Veths) + len(orphans.FirewallRules)

			if totalOrphans > 0 {
				fmt.Printf("Found %d orphaned resource(s):\n", totalOrphans)

				if len(orphans.Veths) > 0 {
					fmt.Printf("  Orphaned veth interfaces (%d):\n", len(orphans.Veths))
					for _, veth := range orphans.Veths {
						fmt.Printf("    - %s\n", veth)
					}
				}

				if len(orphans.FirewallRules) > 0 {
					fmt.Printf("  Orphaned firewall rules (%d):\n", len(orphans.FirewallRules))
					for _, rule := range orphans.FirewallRules {
						fmt.Printf("    - %s\n", rule)
					}
				}

				if !cleanDryRun {
					if !cleanForce {
						fmt.Print("\nClean up orphaned resources? [y/N]: ")
						var response string
						_, _ = fmt.Scanln(&response)
						if response != "y" && response != "Y" {
							fmt.Println("Cancelled.")
							return nil
						}
					}

					logger := func(msg string) {
						fmt.Println(msg)
					}

					if len(orphans.Veths) > 0 {
						vethsCleaned, _ := cleanup.CleanupOrphanedVeths(orphans.Veths, logger)
						cleaned += vethsCleaned
					}

					if len(orphans.FirewallRules) > 0 {
						rulesCleaned, _ := cleanup.CleanupOrphanedFirewallRules(orphans.FirewallRules, logger)
						cleaned += rulesCleaned
					}
				}
			} else {
				fmt.Println("  (no orphaned resources found)")
			}
		}
	}

	if cleanDryRun {
		fmt.Println("\n[Dry run] No changes made.")
		return nil
	}

	if cleaned > 0 {
		fmt.Printf("\n✓ Cleaned %d item(s)\n", cleaned)
	} else {
		fmt.Println("\nNothing to clean.")
	}

	return nil
}
