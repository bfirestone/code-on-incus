package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mensfeld/claude-on-incus/internal/config"
	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/session"
	"github.com/spf13/cobra"
)

var (
	listAll bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active containers and saved sessions",
	Long: `List active claude-on-incus containers and saved sessions.

By default, shows only active containers. Use --all to also show saved sessions.

Examples:
  coi list
  coi list --all
`,
	RunE: listCommand,
}

func init() {
	listCmd.Flags().BoolVar(&listAll, "all", false, "Show saved sessions in addition to active containers")
}

func listCommand(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// List active containers
	fmt.Println("Active Containers:")
	fmt.Println("------------------")

	containers, err := listActiveContainers()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, c := range containers {
			fmt.Printf("  %s\n", c.Name)
			fmt.Printf("    Status: %s\n", c.Status)
			fmt.Printf("    Created: %s\n", c.CreatedAt)
			if c.Image != "" {
				fmt.Printf("    Image: %s\n", c.Image)
			}
		}
	}

	// List saved sessions if --all
	if listAll {
		fmt.Println("\nSaved Sessions:")
		fmt.Println("---------------")

		sessions, err := listSavedSessions(cfg.Paths.SessionsDir)
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		if len(sessions) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, s := range sessions {
				fmt.Printf("  %s\n", s.ID)
				fmt.Printf("    Saved: %s\n", s.SavedAt)
				if s.Workspace != "" {
					fmt.Printf("    Workspace: %s\n", s.Workspace)
				}
			}
		}
	}

	return nil
}

// ContainerInfo holds information about a container
type ContainerInfo struct {
	Name      string
	Status    string
	CreatedAt string
	Image     string
}

// SessionInfo holds information about a saved session
type SessionInfo struct {
	ID        string
	SavedAt   string
	Workspace string
}

// listActiveContainers lists all active claude-on-incus containers
func listActiveContainers() ([]ContainerInfo, error) {
	output, err := container.IncusOutput("list", "^claude-", "--format=json")
	if err != nil {
		return nil, err
	}

	var containers []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &containers); err != nil {
		return nil, err
	}

	var result []ContainerInfo
	for _, c := range containers {
		name, _ := c["name"].(string)
		status, _ := c["status"].(string)
		createdAt, _ := c["created_at"].(string)

		// Get image info
		config, _ := c["config"].(map[string]interface{})
		image, _ := config["image.description"].(string)

		// Parse created_at time
		createdTime := ""
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			createdTime = t.Format("2006-01-02 15:04:05")
		}

		result = append(result, ContainerInfo{
			Name:      name,
			Status:    status,
			CreatedAt: createdTime,
			Image:     image,
		})
	}

	return result, nil
}

// listSavedSessions lists all saved sessions
func listSavedSessions(sessionsDir string) ([]SessionInfo, error) {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionInfo{}, nil
		}
		return nil, err
	}

	var result []SessionInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sessionID := entry.Name()

		// Check if it has a .claude directory
		claudePath := filepath.Join(sessionsDir, sessionID, ".claude")
		if info, err := os.Stat(claudePath); err != nil || !info.IsDir() {
			continue
		}

		// Try to read metadata
		metadataPath := filepath.Join(sessionsDir, sessionID, "metadata.json")
		savedAt := ""
		workspace := ""

		if data, err := os.ReadFile(metadataPath); err == nil {
			var metadata session.SessionMetadata
			if err := json.Unmarshal(data, &metadata); err == nil {
				savedAt = metadata.SavedAt
			}
		}

		// Get directory modification time as fallback
		if savedAt == "" {
			if info, err := entry.Info(); err == nil {
				savedAt = info.ModTime().Format("2006-01-02 15:04:05")
			}
		}

		result = append(result, SessionInfo{
			ID:        sessionID,
			SavedAt:   savedAt,
			Workspace: workspace,
		})
	}

	return result, nil
}
