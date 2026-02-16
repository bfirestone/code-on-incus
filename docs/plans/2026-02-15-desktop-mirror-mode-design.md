# Desktop Mirror Mode Design

Wire up existing but unconnected config fields (`code_user`, `code_uid`, `mount_tool_config`) and add one new field (`workspace_container_path`) so that a COI container can mirror the host desktop environment exactly — same user, same paths, same Claude and tmux configuration.

## Motivation

COI was designed with isolation-first defaults: a `code` user, `/workspace` mount point, and config files copied into the container. This works well for sandboxed agent sessions but prevents Claude's project-specific settings, plugins, skills, statusline scripts, and MCP servers from working correctly — they reference absolute host paths that don't exist in the container.

Desktop mirror mode makes the container behave identically to running `claude` on the host, while still providing COI's container lifecycle, network isolation, resource limits, and tmux orchestration.

## Design Decisions

- **Extend, don't fork.** ~4,000 lines of container management, network isolation, and tmux orchestration are non-trivial to rewrite. The changes are additive (~250 lines) and don't alter default behavior.
- **Wire up existing config, don't invent new abstractions.** `code_user`, `code_uid`, and `mount_tool_config` already exist in config structs or documentation. We finish the wiring.
- **Config-driven activation.** No new CLI flags. Users set preferences once in `config.toml`.
- **Arch-based image.** Matches the host distro for binary compatibility. Shipped as an example build script.
- **Direct mount for `~/.claude`.** Shared state between host and container is a feature — session history, project configs, and credentials stay in sync.

## Configuration

```toml
[defaults]
image = "coi-desktop"
persistent = true
mount_tool_config = true              # mount ~/.claude directly instead of copying

[incus]
code_user = "bfirestone"              # existing field, currently unwired
code_uid = 1000                       # existing field, currently unwired
workspace_container_path = ""         # new: empty = mirror host path, "/workspace" = legacy default

[mounts]
default = [
    { host = "~/.config/tmux", container = "/home/bfirestone/.config/tmux" },
    { host = "~/.tmux", container = "/home/bfirestone/.tmux" },
]
```

### Field semantics

| Field | Default | Behavior |
|---|---|---|
| `mount_tool_config` | `false` | When `true`, mount `~/.claude` and `~/.claude.json` directly into the container. Skip config copying, credential injection, and session save/restore. |
| `code_user` | `"code"` | Username inside the container. Determines home dir (`/home/{code_user}`). Must match a user in the image. |
| `code_uid` | `1000` | UID for command execution and file ownership. |
| `workspace_container_path` | `"/workspace"` | Mount point for the workspace inside the container. Empty string means mirror the host's absolute path. |

## Component Changes

### 1. Config struct additions (`internal/config/config.go`)

Add to `DefaultsConfig`:

```go
type DefaultsConfig struct {
    Image           string `toml:"image"`
    Persistent      bool   `toml:"persistent"`
    Model           string `toml:"model"`
    MountToolConfig bool   `toml:"mount_tool_config"` // NEW
}
```

Add to `IncusConfig`:

```go
type IncusConfig struct {
    Project                string `toml:"project"`
    Group                  string `toml:"group"`
    CodeUID                int    `toml:"code_uid"`
    CodeUser               string `toml:"code_user"`
    DisableShift           bool   `toml:"disable_shift"`
    WorkspaceContainerPath string `toml:"workspace_container_path"` // NEW
}
```

Default `WorkspaceContainerPath` to `"/workspace"` in `DefaultConfig()`. Default `MountToolConfig` to `false`. Add merge logic for both fields.

### 2. Wire up `code_user` and `code_uid` (`internal/session/setup.go`, `cleanup.go`, `cli/shell.go`, `cli/attach.go`, `cli/run.go`)

**Problem:** `container.CodeUser` and `container.CodeUID` are constants in `commands.go:15-17`. The config values exist but are never read. These constants are referenced in 14 places across 5 files.

**Approach:**

Add fields to `SetupOptions`:

```go
type SetupOptions struct {
    // ... existing fields ...
    CodeUser  string // from config, defaults to container.CodeUser
    CodeUID   int    // from config, defaults to container.CodeUID
}
```

Add fields to `SetupResult`:

```go
type SetupResult struct {
    // ... existing fields ...
    CodeUser  string
    CodeUID   int
}
```

In `cli/shell.go`, populate from config with fallback to constants:

```go
codeUser := cfg.Incus.CodeUser
if codeUser == "" {
    codeUser = container.CodeUser
}
codeUID := cfg.Incus.CodeUID
if codeUID == 0 {
    codeUID = container.CodeUID
}
```

All internal references in `setup.go` change from `container.CodeUser` to `opts.CodeUser` / `result.CodeUser`, and from `container.CodeUID` to `opts.CodeUID` / `result.CodeUID`.

In `shell.go`, exec calls change from `container.CodeUID` to `result.CodeUID`.

In `cleanup.go`, `saveSessionData` uses `result.HomeDir` (already available) instead of computing `/home/ + container.CodeUser`.

In `attach.go` and `run.go`, resolve user from config with the same fallback pattern.

The constants remain as fallback defaults. No existing behavior changes unless `code_user` is set in config.

**Files and estimated changes:**

| File | Change |
|---|---|
| `session/setup.go` | Add fields to options/result, replace 8 constant references |
| `session/cleanup.go` | Use passed-in home dir instead of recomputing (1 reference) |
| `cli/shell.go` | Populate from config, use result values (2 references) |
| `cli/attach.go` | Resolve from config (2 references) |
| `cli/run.go` | Resolve from config (1 reference) |

### 3. Configurable workspace container path (`internal/session/setup.go`, `security.go`, `cli/shell.go`, `cli/run.go`, `cli/attach.go`)

**Problem:** `"/workspace"` is hardcoded in 13 places across 6 files as mount target, working directory, and tmux session start path.

**Approach:**

Resolve the workspace container path once in `shell.go`:

```go
workspaceContainerPath := cfg.Incus.WorkspaceContainerPath
if workspaceContainerPath == "" {
    workspaceContainerPath = absWorkspace // mirror host path
}
```

Add `WorkspaceContainerPath` to `SetupOptions` and `SetupResult`. All downstream code uses `result.WorkspaceContainerPath` instead of the string literal `"/workspace"`.

In `setup.go`, the mount call changes:

```go
// Before:
mgr.MountDisk("workspace", opts.WorkspacePath, "/workspace", useShift, false)
// After:
mgr.MountDisk("workspace", opts.WorkspacePath, opts.WorkspaceContainerPath, useShift, false)
```

In `shell.go`, all `Cwd: "/workspace"` and tmux `-c /workspace` become `Cwd: result.WorkspaceContainerPath`.

In `security.go`, `SetupSecurityMounts` gets a new `containerWorkspacePath` parameter:

```go
// Before:
containerPath := filepath.Join("/workspace", relPath)
// After:
containerPath := filepath.Join(containerWorkspacePath, relPath)
```

In `commands.go`, the default `Cwd` fallback stays as `"/workspace"` — it's only hit if no Cwd is provided, and we'll always provide one in the updated code paths.

**Files and references to update:**

| File | `"/workspace"` occurrences |
|---|---|
| `session/setup.go` | 1 (mount target) |
| `session/security.go` | 1 (path join) |
| `cli/shell.go` | 8 (Cwd fields + tmux `-c`) |
| `cli/run.go` | 2 (mount + exec cwd) |
| `cli/attach.go` | 2 (Cwd fields) |

### 4. `mount_tool_config` — mount instead of copy (`internal/session/setup.go`, `cleanup.go`, `cli/shell.go`)

**Problem:** When `mount_tool_config = true`, the entire `~/.claude` directory is mounted directly. The config copy pipeline (`setupCLIConfig`), credential injection (`injectCredentials`), session save (`saveSessionData`), and session restore (`restoreSessionData`) must all be skipped.

**Approach:**

Add `MountToolConfig bool` to `SetupOptions`. In `shell.go`, populate from `cfg.Defaults.MountToolConfig`.

In `setup.go`, step 11 (CLI tool config):

```go
if opts.MountToolConfig && opts.CLIConfigPath != "" {
    // Mount host config dir directly
    containerConfigPath := filepath.Join(result.HomeDir, configDirName)
    mgr.MountDisk("tool-config", opts.CLIConfigPath, containerConfigPath, useShift, false)

    // Also mount the state file (~/.claude.json)
    stateFile := fmt.Sprintf(".%s.json", opts.Tool.Name())
    hostStateFile := filepath.Join(filepath.Dir(opts.CLIConfigPath), stateFile)
    if _, err := os.Stat(hostStateFile); err == nil {
        containerStateFile := filepath.Join(result.HomeDir, stateFile)
        mgr.MountDisk("tool-state", hostStateFile, containerStateFile, useShift, false)
    }
} else {
    // Existing behavior: copy essential files
    setupCLIConfig(...)
}
```

In `setup.go`, step 9 (resume path) — wrap with `!opts.MountToolConfig`:

```go
if opts.ResumeFromID != "" && opts.Tool != nil && opts.Tool.ConfigDirName() != "" && !opts.MountToolConfig {
    // existing restore + credential injection logic
}
```

In `cleanup.go` — `saveSessionData` needs to know about mount mode. Add `MountToolConfig bool` parameter. When true, skip the save:

```go
if mountToolConfig {
    logger("Tool config is directly mounted, skipping session save")
    return nil
}
```

Store `MountToolConfig` in `SetupResult` so cleanup has access.

### 5. Arch-based desktop build script (`scripts/build/desktop-arch.sh`)

A complete build script using `pacman` instead of `apt`, starting from `images:archlinux`. Creates user directly (no `ubuntu` user rename). Same structural flow as `coi.sh`.

```bash
#!/bin/bash
set -euo pipefail

CODE_USER="bfirestone"
CODE_UID=1000

log() { echo "[coi-desktop] $*"; }

configure_dns_if_needed() {
    log "Checking DNS configuration..."
    if getent hosts archlinux.org > /dev/null 2>&1; then
        log "DNS resolution works."
        return 0
    fi
    log "DNS resolution failed, configuring static DNS..."
    rm -f /etc/resolv.conf
    cat > /etc/resolv.conf << 'EOF'
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 1.1.1.1
EOF
    log "Static DNS configured."
}

install_base_dependencies() {
    log "Installing base dependencies..."
    pacman -Syu --noconfirm
    pacman -S --noconfirm --needed \
        curl wget git ca-certificates jq unzip sudo \
        tmux base-devel openssl readline zlib \
        libffi libyaml gmp sqlite postgresql-libs \
        libxml2 libxslt docker docker-compose \
        github-cli nodejs npm python
}

create_user() {
    log "Creating user $CODE_USER..."
    useradd -m -u "$CODE_UID" -G wheel,docker "$CODE_USER"
    echo "$CODE_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/"$CODE_USER"
    chmod 440 /etc/sudoers.d/"$CODE_USER"
    mkdir -p "/home/$CODE_USER/.claude" "/home/$CODE_USER/.ssh"
    chmod 700 "/home/$CODE_USER/.ssh"
    chown -R "$CODE_USER:$CODE_USER" "/home/$CODE_USER"
    log "User '$CODE_USER' created (uid: $CODE_UID)"
}

configure_power_wrappers() {
    log "Configuring power management wrappers..."
    for cmd in shutdown poweroff reboot halt; do
        cat > "/usr/local/bin/${cmd}" << 'WRAPPER_EOF'
#!/bin/bash
exec sudo /usr/sbin/COMMAND_NAME "$@"
WRAPPER_EOF
        sed -i "s/COMMAND_NAME/${cmd}/" "/usr/local/bin/${cmd}"
        chmod 755 "/usr/local/bin/${cmd}"
    done
}

install_claude_cli() {
    log "Installing Claude CLI..."
    su - "$CODE_USER" -c 'curl -fsSL https://claude.ai/install.sh | bash'
    local CLAUDE_PATH="/home/$CODE_USER/.local/bin/claude"
    if [[ ! -x "$CLAUDE_PATH" ]]; then
        log "ERROR: Claude CLI not found at $CLAUDE_PATH"
        exit 1
    fi
    ln -sf "$CLAUDE_PATH" /usr/local/bin/claude
    log "Claude CLI installed"
}

install_dummy() {
    log "Installing dummy..."
    if [[ -f /tmp/dummy ]]; then
        cp /tmp/dummy /usr/local/bin/dummy
        chmod +x /usr/local/bin/dummy
        rm /tmp/dummy
    else
        log "No dummy found, skipping (optional for custom builds)"
    fi
}

cleanup() {
    log "Cleaning up..."
    pacman -Scc --noconfirm
}

main() {
    log "Starting coi-desktop image build (Arch Linux)..."
    configure_dns_if_needed
    install_base_dependencies
    create_user
    configure_power_wrappers
    install_claude_cli
    install_dummy
    cleanup
    log "coi-desktop image build complete!"
}

main "$@"
```

Build with:

```bash
coi build custom coi-desktop --base images:archlinux --script scripts/build/desktop-arch.sh
```

### 6. Validation and error handling

**User-exists check:** After container start, verify the configured user exists in the container. If not, fail with a clear message suggesting the custom image build command.

```go
// In setup.go, after container is ready:
if !result.RunAsRoot {
    checkCmd := fmt.Sprintf("id %s", opts.CodeUser)
    if _, err := mgr.ExecCommand(checkCmd, container.ExecCommandOptions{Capture: true}); err != nil {
        return nil, fmt.Errorf(
            "user '%s' does not exist in image '%s' - build a custom image with this user:\n"+
            "  coi build custom coi-desktop --base images:archlinux --script scripts/build/desktop-arch.sh",
            opts.CodeUser, image,
        )
    }
}
```

**Path mismatch warning:** When `mount_tool_config = true`, warn if host home differs from container home:

```go
hostHome, _ := os.UserHomeDir()
if opts.MountToolConfig && hostHome != result.HomeDir {
    opts.Logger(fmt.Sprintf(
        "Warning: mount_tool_config is enabled but host home (%s) differs from container home (%s). "+
        "Absolute paths in tool config may not resolve correctly. Consider setting code_user to match.",
        hostHome, result.HomeDir,
    ))
}
```

**Session save skip:** When `mount_tool_config = true`, skip `saveSessionData` in cleanup. The tool config is live on the host mount.

**Concurrent access:** No code change. Claude handles concurrent access to `~/.claude` via per-project subdirs and unique session files. Documented as a known consideration.

## Files Changed Summary

| File | Change | Est. lines |
|---|---|---|
| `internal/config/config.go` | Add `MountToolConfig`, `WorkspaceContainerPath` fields, defaults, merge | ~20 |
| `internal/session/setup.go` | Add fields to options/result, conditional mount-vs-copy, config user/uid, workspace path, validation | ~40 |
| `internal/session/cleanup.go` | Skip save when mounted, use passed-in home dir | ~10 |
| `internal/session/security.go` | Accept workspace container path parameter | ~5 |
| `internal/cli/shell.go` | Resolve config values, populate options, use result for Cwd/user | ~30 |
| `internal/cli/run.go` | Use config for user/uid and workspace path | ~15 |
| `internal/cli/attach.go` | Use config for user/uid and workspace path | ~10 |
| `scripts/build/desktop-arch.sh` | Arch-based desktop build script | ~100 |
| **Total** | | **~250** |

## Usage

### One-time setup

```bash
# 1. Build the desktop image
coi build custom coi-desktop --base images:archlinux --script scripts/build/desktop-arch.sh

# 2. Configure (~/.config/coi/config.toml)
cat >> ~/.config/coi/config.toml << 'EOF'
[defaults]
image = "coi-desktop"
persistent = true
mount_tool_config = true

[incus]
code_user = "bfirestone"
code_uid = 1000
workspace_container_path = ""

[mounts]
default = [
    { host = "~/.config/tmux", container = "/home/bfirestone/.config/tmux" },
    { host = "~/.tmux", container = "/home/bfirestone/.tmux" },
]
EOF
```

### Daily use

```bash
# Work in a single repo (mounted at its real host path)
cd ~/devspace/personal/github/code-on-incus
coi shell

# Work across multiple repos
coi shell -w ~/devspace

# Everything works: tmux theme, Claude statusline, plugins, project configs, MCP servers
```

## Backwards Compatibility

All defaults match current behavior. No existing user is affected unless they explicitly set the new config values.

| Config | Default | Current behavior preserved? |
|---|---|---|
| `mount_tool_config` | `false` | Yes — config files are copied as before |
| `code_user` | `"code"` | Yes — existing constant value |
| `code_uid` | `1000` | Yes — existing constant value |
| `workspace_container_path` | `"/workspace"` | Yes — existing hardcoded value |
