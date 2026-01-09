package image

import (
	"fmt"
	"os"

	"github.com/mensfeld/claude-on-incus/internal/container"
)

// installBaseDependencies installs base packages and Node.js
func (b *Builder) installBaseDependencies() error {
	b.opts.Logger("Installing base dependencies...")

	// Update package list
	if err := b.execInContainer("apt-get update -qq", false); err != nil {
		return err
	}

	// Install base packages
	packages := []string{
		"curl", "wget", "git", "ca-certificates", "gnupg", "jq", "unzip", "sudo",
		"tmux", // For session management and background processes
		"build-essential", "libssl-dev", "libreadline-dev", "zlib1g-dev",
		"libffi-dev", "libyaml-dev", "libgmp-dev",
		"libsqlite3-dev", "libpq-dev", "libmysqlclient-dev",
		"libxml2-dev", "libxslt1-dev", "libcurl4-openssl-dev",
	}

	cmd := fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get install -y -qq %s", joinStrings(packages, " "))
	if err := b.execInContainer(cmd, false); err != nil {
		return err
	}

	// Install Node.js 20 LTS
	b.opts.Logger("Installing Node.js LTS...")
	if err := b.execInContainer("curl -fsSL https://deb.nodesource.com/setup_20.x | bash -", false); err != nil {
		return err
	}

	if err := b.execInContainer("apt-get install -y -qq nodejs", false); err != nil {
		return err
	}

	// Verify Node.js installation
	output, err := b.mgr.ExecCommand("node --version", container.ExecCommandOptions{Capture: true})
	if err != nil {
		return err
	}
	b.opts.Logger(fmt.Sprintf("Node.js: %s", output))

	return nil
}

// createClaudeUser creates the claude user with passwordless sudo
func (b *Builder) createClaudeUser(privileged bool) error {
	b.opts.Logger("Creating claude user...")

	// Rename ubuntu user to claude
	commands := []string{
		fmt.Sprintf("usermod -l %s -d /home/%s -m ubuntu", ClaudeUser, ClaudeUser),
		fmt.Sprintf("groupmod -n %s ubuntu", ClaudeUser),
		fmt.Sprintf("mkdir -p /home/%s/.claude", ClaudeUser),
		fmt.Sprintf("chown -R %s:%s /home/%s", ClaudeUser, ClaudeUser, ClaudeUser),
	}

	for _, cmd := range commands {
		if err := b.execInContainer(cmd, false); err != nil {
			return err
		}
	}

	// Setup passwordless sudo
	sudoersContent := fmt.Sprintf("%s ALL=(ALL) NOPASSWD:ALL", ClaudeUser)
	if err := b.mgr.CreateFile(fmt.Sprintf("/etc/sudoers.d/%s", ClaudeUser), sudoersContent); err != nil {
		return err
	}

	// Fix ownership and permissions (sudoers files must be owned by root)
	if err := b.execInContainer(fmt.Sprintf("chown root:root /etc/sudoers.d/%s", ClaudeUser), false); err != nil {
		return err
	}

	if err := b.execInContainer(fmt.Sprintf("chmod 440 /etc/sudoers.d/%s", ClaudeUser), false); err != nil {
		return err
	}

	if err := b.execInContainer(fmt.Sprintf("usermod -aG sudo %s", ClaudeUser), false); err != nil {
		return err
	}

	b.opts.Logger(fmt.Sprintf("User '%s' created with passwordless sudo (uid: %d)", ClaudeUser, ClaudeUID))
	return nil
}

// installClaudeCLI installs the Claude CLI from npm
func (b *Builder) installClaudeCLI() error {
	b.opts.Logger("Installing Claude CLI...")

	if err := b.execInContainer("npm install -g @anthropic-ai/claude-code", false); err != nil {
		return err
	}

	// Verify installation
	output, err := b.mgr.ExecCommand("claude --version", container.ExecCommandOptions{Capture: true})
	if err == nil {
		b.opts.Logger(fmt.Sprintf("Claude CLI: %s", output))
	}

	return nil
}

// installDocker installs Docker CE
func (b *Builder) installDocker() error {
	b.opts.Logger("Installing Docker...")

	// Create keyrings directory
	if err := b.execInContainer("install -m 0755 -d /etc/apt/keyrings", false); err != nil {
		return err
	}

	// Add Docker GPG key
	if err := b.execInContainer("curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg", false); err != nil {
		return err
	}

	if err := b.execInContainer("chmod a+r /etc/apt/keyrings/docker.gpg", false); err != nil {
		return err
	}

	// Add Docker repository
	repoCmd := `echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null`
	if err := b.execInContainer(repoCmd, false); err != nil {
		return err
	}

	// Update and install Docker
	if err := b.execInContainer("apt-get update -qq", false); err != nil {
		return err
	}

	dockerPackages := []string{
		"docker-ce",
		"docker-ce-cli",
		"containerd.io",
		"docker-buildx-plugin",
		"docker-compose-plugin",
	}
	cmd := fmt.Sprintf("DEBIAN_FRONTEND=noninteractive apt-get install -y -qq %s", joinStrings(dockerPackages, " "))
	if err := b.execInContainer(cmd, false); err != nil {
		return err
	}

	// Add claude user to docker group
	if err := b.execInContainer(fmt.Sprintf("usermod -aG docker %s", ClaudeUser), false); err != nil {
		return err
	}

	// Verify Docker installation
	output, err := b.mgr.ExecCommand("docker --version", container.ExecCommandOptions{Capture: true})
	if err == nil {
		b.opts.Logger(fmt.Sprintf("Docker: %s", output))
	}

	return nil
}

// installTestClaude installs the fake Claude CLI as test-claude for testing
func (b *Builder) installTestClaude() error {
	b.opts.Logger("Installing test-claude (fake Claude for testing)...")

	// Path to fake Claude script on the host
	// Try to find it relative to the current working directory
	fakeClaudeHostPath := "testdata/fake-claude/claude"

	// Check if file exists on host
	if _, err := os.Stat(fakeClaudeHostPath); err != nil {
		b.opts.Logger("Warning: fake Claude not found on host, skipping test-claude installation")
		return nil // Non-fatal, just skip
	}

	// Push fake Claude script to container
	if err := b.mgr.PushFile(fakeClaudeHostPath, "/usr/local/bin/test-claude"); err != nil {
		b.opts.Logger("Warning: failed to push test-claude, skipping")
		return nil // Non-fatal, just skip
	}

	// Make executable
	if err := b.execInContainer("chmod +x /usr/local/bin/test-claude", false); err != nil {
		return err
	}

	// Verify installation
	output, err := b.mgr.ExecCommand("test-claude --version", container.ExecCommandOptions{Capture: true})
	if err == nil {
		b.opts.Logger(fmt.Sprintf("test-claude: %s", output))
	}

	b.opts.Logger("test-claude installed successfully (use COI_USE_TEST_CLAUDE=1 to enable)")
	return nil
}

// joinStrings joins a slice of strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
