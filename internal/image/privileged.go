package image

import (
	"fmt"

	"github.com/mensfeld/claude-on-incus/internal/container"
)

// installGitHubCLI installs the GitHub CLI (gh)
func (b *Builder) installGitHubCLI() error {
	b.opts.Logger("Installing GitHub CLI...")

	// Add GitHub CLI GPG key
	if err := b.execInContainer("curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg", false); err != nil {
		return err
	}

	if err := b.execInContainer("chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg", false); err != nil {
		return err
	}

	// Add GitHub CLI repository
	repoCmd := `echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null`
	if err := b.execInContainer(repoCmd, false); err != nil {
		return err
	}

	// Update and install GitHub CLI
	if err := b.execInContainer("apt-get update -qq", false); err != nil {
		return err
	}

	if err := b.execInContainer("DEBIAN_FRONTEND=noninteractive apt-get install -y -qq gh", false); err != nil {
		return err
	}

	// Verify installation
	output, err := b.mgr.ExecCommand("gh --version", container.ExecCommandOptions{Capture: true})
	if err == nil {
		// Get just the first line (version)
		lines := splitLines(output)
		if len(lines) > 0 {
			b.opts.Logger(fmt.Sprintf("GitHub CLI: %s", lines[0]))
		}
	}

	return nil
}

// setupSSHDirectory creates and configures the SSH directory
func (b *Builder) setupSSHDirectory() error {
	b.opts.Logger("Setting up SSH directory...")

	commands := []string{
		fmt.Sprintf("mkdir -p /home/%s/.ssh", ClaudeUser),
		fmt.Sprintf("chmod 700 /home/%s/.ssh", ClaudeUser),
		fmt.Sprintf("chown -R %s:%s /home/%s/.ssh", ClaudeUser, ClaudeUser, ClaudeUser),
	}

	for _, cmd := range commands {
		if err := b.execInContainer(cmd, false); err != nil {
			return err
		}
	}

	return nil
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
