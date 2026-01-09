# claude-on-incus Enhancement Requirements

## Purpose

This document outlines the enhancements needed for **claude-on-incus** to serve as a low-level Incus abstraction layer for **claude_yard** (a workflow orchestration platform that runs Claude Code sessions in containers).

Currently, `coi` is a high-level CLI tool focused on interactive Claude sessions. claude_yard needs programmatic access to the underlying container operations that `coi` already provides internally.

---

## Current State vs. Required State

### What coi Currently Provides

✅ Internal packages with full container lifecycle management (`internal/container/manager.go`)
✅ Session management with .claude directory persistence (`internal/session/`)
✅ Image building for sandbox and privileged images (`internal/image/builder.go`)
✅ Configuration system (`internal/config/`)

### What coi CLI Currently Exposes

✅ `coi shell` - Interactive Claude session
✅ `coi build sandbox|privileged` - Build system images
✅ `coi list` - List containers/sessions
✅ `coi images` - List images (no filtering)
✅ `coi clean` - Clean stopped containers

### What's Missing

❌ **Low-level container operation commands** (launch, stop, exec, mount)
❌ **File operation commands** (push, pull)
❌ **Image management commands** (publish, delete, exists, cleanup)
❌ **Custom image building** from user-provided scripts
❌ **Image versioning utilities** (list by prefix, cleanup old versions)
❌ **Structured output** (JSON format for programmatic parsing)

---

## Required Enhancements

### 1. Container Operations Commands

Add new CLI command group: `coi container <subcommand>`

#### 1.1 Launch Container

```bash
coi container launch <image> <name> [--ephemeral] [--project PROJECT]
```

**Description**: Launch a new container from an image.

**Arguments**:
- `<image>` - Image alias or fingerprint to launch from
- `<name>` - Container name

**Flags**:
- `--ephemeral` - Make container ephemeral (default: false)
- `--project` - Incus project (default: "default")

**Output**: Success/error message
**Exit Code**: 0 on success, non-zero on error

**Implementation**: Call `container.Manager.Launch()`

---

#### 1.2 Start Container

```bash
coi container start <name>
```

**Description**: Start a stopped container.

**Implementation**: Call `container.Manager.Start()`

---

#### 1.3 Stop Container

```bash
coi container stop <name> [--force]
```

**Description**: Stop a running container.

**Flags**:
- `--force` - Force stop (default: false)

**Implementation**: Call `container.Manager.Stop()`

---

#### 1.4 Delete Container

```bash
coi container delete <name> [--force]
```

**Description**: Delete a container.

**Flags**:
- `--force` - Force delete even if running (default: false)

**Implementation**: Call `container.Manager.Delete()`

---

#### 1.5 Execute Command in Container

```bash
coi container exec <name> [--user UID] [--group GID] [--env KEY=VAL]... [--cwd PATH] -- <command>
```

**Description**: Execute a command inside a container with full context control.

**Arguments**:
- `<name>` - Container name
- `<command>` - Command to execute (after `--`)

**Flags**:
- `--user UID` - User ID to run as
- `--group GID` - Group ID to run as (defaults to user if not specified)
- `--env KEY=VAL` - Environment variable (repeatable)
- `--cwd PATH` - Working directory (default: "/workspace")
- `--capture` - Capture and return output (vs. streaming to stdout)

**Output**:
- Without `--capture`: Command output streams to stdout/stderr
- With `--capture`: JSON with stdout, stderr, exit code

**Exit Code**: Exit code of the executed command

**Implementation**: Call `container.Manager.ExecCommand()` with `ExecCommandOptions`

**Example**:
```bash
# Run as root
coi container exec my-container -- ls -la /

# Run as specific user with env vars
coi container exec my-container --user 1000 --env FOO=bar --cwd /workspace -- npm test

# Capture output
coi container exec my-container --capture -- echo "hello world"
# Output: {"stdout": "hello world\n", "stderr": "", "exit_code": 0}
```

---

#### 1.6 Check if Container Exists

```bash
coi container exists <name>
```

**Description**: Check if a container exists (running or stopped).

**Output**: No output (use exit code)
**Exit Code**: 0 if exists, 1 if not

**Implementation**: Call `container.Manager.Exists()`

---

#### 1.7 Check if Container is Running

```bash
coi container running <name>
```

**Description**: Check if a container is currently running.

**Output**: No output (use exit code)
**Exit Code**: 0 if running, 1 if not running or doesn't exist

**Implementation**: Call `container.Manager.Running()`

---

#### 1.8 Mount Disk to Container

```bash
coi container mount <name> <device-name> <source> <path> [--shift]
```

**Description**: Add a disk device to a container.

**Arguments**:
- `<name>` - Container name
- `<device-name>` - Device name (e.g., "workspace", "storage")
- `<source>` - Host path to mount
- `<path>` - Container path to mount at

**Flags**:
- `--shift` - Enable UID/GID shifting (default: true)

**Implementation**: Call `container.Manager.MountDisk()`

**Example**:
```bash
coi container mount my-container workspace /home/user/project /workspace --shift
```

---

### 2. File Operations Commands

Add new CLI command group: `coi file <subcommand>`

#### 2.1 Push File/Directory

```bash
coi file push <local-path> <container>:<remote-path> [-r]
```

**Description**: Push a file or directory into a container.

**Arguments**:
- `<local-path>` - Local file or directory path
- `<container>:<remote-path>` - Destination in format "container-name:/path/in/container"

**Flags**:
- `-r, --recursive` - Push directory recursively (default: false)

**Implementation**:
- File: Call `container.Manager.PushFile()`
- Directory: Call `container.Manager.PushDirectory()`

**Examples**:
```bash
# Push file
coi file push ./config.json my-container:/workspace/config.json

# Push directory
coi file push -r ./src my-container:/workspace/src
```

---

#### 2.2 Pull File/Directory

```bash
coi file pull <container>:<remote-path> <local-path> [-r]
```

**Description**: Pull a file or directory from a container.

**Arguments**:
- `<container>:<remote-path>` - Source in format "container-name:/path/in/container"
- `<local-path>` - Local destination path

**Flags**:
- `-r, --recursive` - Pull directory recursively (default: false)

**Implementation**: Call `container.Manager.PullDirectory()`

**Example**:
```bash
# Pull directory (e.g., save Claude session data)
coi file pull -r my-container:/root/.claude ./saved-sessions/session-123/
```

---

### 3. Image Operations Commands

Extend existing `coi image` command group with new subcommands.

#### 3.1 Publish Container as Image

```bash
coi image publish <container> <alias> [--description TEXT]
```

**Description**: Publish a stopped container as an image.

**Arguments**:
- `<container>` - Container name to publish
- `<alias>` - Alias for the new image

**Flags**:
- `--description TEXT` - Image description (optional)

**Output**: JSON with fingerprint and alias
```json
{
  "fingerprint": "abc123def456...",
  "alias": "my-custom-image"
}
```

**Exit Code**: 0 on success, non-zero on error

**Implementation**:
1. Stop container if running (`container.StopContainer()`)
2. Publish (`container.PublishContainer()`)
3. Return fingerprint and alias

**Example**:
```bash
coi image publish my-container my-image --description "Custom build with Python 3.11"
# Output: {"fingerprint": "abc123...", "alias": "my-image"}
```

---

#### 3.2 Delete Image

```bash
coi image delete <alias>
```

**Description**: Delete an image by alias.

**Arguments**:
- `<alias>` - Image alias to delete

**Exit Code**: 0 on success, non-zero on error

**Implementation**: Call `container.DeleteImage()`

---

#### 3.3 Check if Image Exists

```bash
coi image exists <alias>
```

**Description**: Check if an image with the given alias exists.

**Arguments**:
- `<alias>` - Image alias to check

**Output**: No output (use exit code)
**Exit Code**: 0 if exists, 1 if not

**Implementation**: Call `container.ImageExists()`

---

#### 3.4 List Images with Filtering

```bash
coi image list [--prefix TEXT] [--format json|table]
```

**Description**: List available images, optionally filtered by alias prefix.

**Flags**:
- `--prefix TEXT` - Only show images whose alias starts with this prefix
- `--format json|table` - Output format (default: table)

**Output**:
- Table format: Human-readable table
- JSON format: Array of image objects
```json
[
  {
    "fingerprint": "abc123...",
    "aliases": ["my-image", "my-image-v1"],
    "size": 1234567890,
    "created_at": "2026-01-08T10:30:00Z"
  }
]
```

**Implementation**:
1. Call `container.IncusOutput("image", "list", "--format=json")`
2. Parse JSON
3. Filter by prefix if specified
4. Format output

**Example**:
```bash
# List all images
coi image list

# List images for a specific node (versioned images)
coi image list --prefix claudeyard-node-42- --format json
# Output: [{"fingerprint": "...", "aliases": ["claudeyard-node-42-20260108-103000"]}, ...]
```

---

#### 3.5 Cleanup Old Image Versions

```bash
coi image cleanup <prefix> --keep <N>
```

**Description**: Delete old versions of images matching a prefix, keeping only the N most recent.

**Arguments**:
- `<prefix>` - Image alias prefix (e.g., "claudeyard-node-42-")

**Flags**:
- `--keep N` - Number of versions to keep (required)

**Logic**:
1. List all images matching prefix
2. Sort by timestamp extracted from alias (format: `prefix-YYYYMMDD-HHMMSS`)
3. Keep the N most recent
4. Delete the rest

**Output**: List of deleted aliases

**Exit Code**: 0 on success, non-zero on error

**Implementation**: New function in `internal/image/versions.go`

**Example**:
```bash
# Keep only the 3 most recent versions of node-42 images
coi image cleanup claudeyard-node-42- --keep 3
# Output:
# Deleted: claudeyard-node-42-20260101-120000
# Deleted: claudeyard-node-42-20260102-140000
# Kept: claudeyard-node-42-20260107-160000
# Kept: claudeyard-node-42-20260108-100000
# Kept: claudeyard-node-42-20260108-103000
```

---

### 4. Custom Image Building

Extend `coi build` command with custom image support.

#### 4.1 Build Custom Image

```bash
coi build custom <name> --base <base-image> --script <build-script.sh> [--privileged]
```

**Description**: Build a custom image from a base image using a user-provided build script.

**Arguments**:
- `<name>` - Alias for the new custom image

**Flags**:
- `--base IMAGE` - Base image to build from (default: "coi-sandbox")
- `--script PATH` - Path to build script (bash script to run in container)
- `--privileged` - Use privileged base if base not specified (default: false)

**Build Process**:
1. Determine base image:
   - If `--base` specified, use that
   - Else if `--privileged`, use "coi-privileged"
   - Else use "coi-sandbox"
2. Launch build container from base image
3. Wait for network
4. Push build script to `/tmp/build.sh`
5. Execute build script as root
6. Stop container
7. Publish container as image with specified name
8. Delete build container

**Output**: Progress messages + final image info (JSON)

**Exit Code**: 0 on success, non-zero on error

**Implementation**: Extend `internal/image/builder.go`

**Example**:
```bash
# Build custom image with Rust toolchain
cat > build-rust.sh <<'EOF'
#!/bin/bash
set -e
apt-get update
apt-get install -y curl build-essential
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
EOF

coi build custom my-rust-image --base coi-sandbox --script build-rust.sh
# Output:
# Building custom image 'my-rust-image' from 'coi-sandbox'...
# Launching build container...
# Running build script...
# Publishing image...
# {"fingerprint": "xyz789...", "alias": "my-rust-image"}
```

---

### 5. General Output Format Requirements

#### 5.1 JSON Output for Programmatic Use

All commands that output data should support `--format json` flag for easy parsing:

- `coi image list --format json`
- `coi container exec --capture` (always JSON)
- `coi image publish` (always JSON)

#### 5.2 Exit Codes

Use standard exit codes for programmatic use:

- **0** - Success
- **1** - General error
- **2** - Command line usage error
- **3** - Container/image not found
- **4** - Permission denied
- **5** - Incus not available

#### 5.3 Boolean Commands

Commands that check state should use exit codes only (no output):

- `coi container exists <name>` - Exit 0 if exists, 1 if not
- `coi container running <name>` - Exit 0 if running, 1 if not
- `coi image exists <alias>` - Exit 0 if exists, 1 if not

This allows shell usage: `if coi container running my-container; then ...`

---

### 6. Image Version Management

Add `internal/image/versions.go` with utilities for managing versioned images.

#### 6.1 ListVersions Function

```go
// ListVersions returns all images matching a prefix, sorted by timestamp
// Assumes aliases follow format: prefix-YYYYMMDD-HHMMSS
func ListVersions(prefix string) ([]string, error)
```

**Logic**:
1. Call `IncusOutput("image", "list", "--format=json")`
2. Parse JSON response
3. Extract aliases matching prefix
4. Sort by timestamp (parsed from alias)
5. Return sorted list

#### 6.2 Cleanup Function

```go
// Cleanup deletes old versions, keeping only the N most recent
func Cleanup(prefix string, keepCount int) error
```

**Logic**:
1. Call `ListVersions(prefix)`
2. Keep the `keepCount` most recent
3. Delete the rest using `DeleteImage()`

#### 6.3 ExtractTimestamp Helper

```go
// ExtractTimestamp parses timestamp from alias like "prefix-20260108-103000"
func ExtractTimestamp(alias string) (time.Time, error)
```

---

## Use Cases

### Use Case 1: Run Custom Workflow Node

claude_yard needs to:
1. Launch ephemeral container from custom image
2. Mount workspace directory
3. Execute Claude CLI with specific prompt
4. Save session data (.claude directory)
5. Cleanup container

**Current approach** (direct Incus):
```bash
incus launch my-image my-container --ephemeral
incus config device add my-container workspace disk source=/path path=/workspace shift=true
incus exec my-container -- bash -c "claude --prompt '...'"
incus file pull -r my-container:/root/.claude /saved-sessions/
incus stop my-container --force
```

**Desired approach** (via coi):
```bash
coi container launch my-image my-container --ephemeral
coi container mount my-container workspace /path /workspace --shift
coi container exec my-container -- bash -c "claude --prompt '...'"
coi file pull -r my-container:/root/.claude /saved-sessions/
coi container stop my-container --force
```

---

### Use Case 2: Persist Node State

claude_yard needs to:
1. Stop container after successful run
2. Publish container as versioned image
3. Cleanup old versions (keep 3)

**Current approach**:
```bash
incus stop my-container
incus publish my-container --alias claudeyard-node-42-20260108-103000 description="..."
# Manual cleanup of old versions
```

**Desired approach**:
```bash
coi container stop my-container
coi image publish my-container claudeyard-node-42-20260108-103000 --description "..."
coi image cleanup claudeyard-node-42- --keep 3
```

---

### Use Case 3: Build Custom Image via Web UI

User provides build script via claude_yard web UI:

```bash
#!/bin/bash
apt-get update
apt-get install -y postgresql-client redis-tools
```

claude_yard needs to build this into an image:

**Desired approach**:
```bash
coi build custom user-custom-db-tools --base coi-privileged --script /tmp/build-script.sh
```

---

## Testing Requirements

### Unit Tests

Add tests for all new CLI commands:

```go
func TestContainerLaunchCmd(t *testing.T) {
    // Test launching ephemeral container
    // Test launching persistent container
    // Test error handling (image not found)
}

func TestContainerExecCmd(t *testing.T) {
    // Test basic execution
    // Test with --user flag
    // Test with --env flags
    // Test with --capture
}

func TestImagePublishCmd(t *testing.T) {
    // Test publishing container
    // Test with description
    // Test JSON output format
}

func TestImageCleanupCmd(t *testing.T) {
    // Test cleanup with multiple versions
    // Test keeping N versions
    // Test timestamp parsing
}
```

### Integration Tests

Test full workflows:

1. Launch → exec → stop → delete
2. Launch → exec → publish → cleanup
3. Custom image build → launch → verify

---

## Documentation Requirements

### 1. README Updates

Add section: **"Using coi as a Container Library"**

Explain that coi can be used as a low-level Incus wrapper for other applications, not just for interactive Claude sessions.

### 2. API Reference Document

Create `docs/API.md` with:

- Complete command reference
- All flags and arguments
- Output format specifications
- Exit code meanings
- Examples for each command

### 3. Migration Guide

Create `docs/MIGRATION.md` explaining how to migrate from direct Incus CLI usage to coi commands.

---

## Implementation Checklist

### Phase 1: Container Operations
- [ ] Add `internal/cli/container.go`
- [ ] Implement `container launch` command
- [ ] Implement `container start` command
- [ ] Implement `container stop` command
- [ ] Implement `container delete` command
- [ ] Implement `container exec` command with all flags
- [ ] Implement `container exists` command
- [ ] Implement `container running` command
- [ ] Implement `container mount` command
- [ ] Add unit tests for container commands
- [ ] Add integration tests for container lifecycle

### Phase 2: File Operations
- [ ] Add `internal/cli/file.go`
- [ ] Implement `file push` command (single file)
- [ ] Implement `file push -r` command (directory)
- [ ] Implement `file pull -r` command
- [ ] Add unit tests for file operations
- [ ] Add integration tests for push/pull

### Phase 3: Image Operations
- [ ] Extend `internal/cli/images.go`
- [ ] Implement `image publish` command with JSON output
- [ ] Implement `image delete` command
- [ ] Implement `image exists` command
- [ ] Extend `image list` with `--prefix` filter
- [ ] Extend `image list` with `--format json`
- [ ] Add `internal/image/versions.go` module
- [ ] Implement `ListVersions()` function
- [ ] Implement `Cleanup()` function
- [ ] Implement `ExtractTimestamp()` helper
- [ ] Implement `image cleanup` command
- [ ] Add unit tests for image operations
- [ ] Add integration tests for publish/cleanup

### Phase 4: Custom Image Building
- [ ] Extend `internal/cli/build.go`
- [ ] Implement `build custom` command
- [ ] Support `--base`, `--script`, `--privileged` flags
- [ ] Add integration tests for custom builds
- [ ] Test with sample build scripts

### Phase 5: Output Formats & Exit Codes
- [ ] Implement `--format json` support for all list commands
- [ ] Implement consistent exit codes (0-5)
- [ ] Ensure boolean commands use exit codes only
- [ ] Add `--capture` support for exec command

### Phase 6: Documentation
- [ ] Update README with "Container Library" section
- [ ] Create `docs/API.md` with full command reference
- [ ] Create `docs/MIGRATION.md` for users migrating from Incus
- [ ] Add examples directory with sample scripts
- [ ] Update CHANGELOG

### Phase 7: Release
- [ ] Tag version (e.g., v1.0.0)
- [ ] Create GitHub release with binaries
- [ ] Update installation script

---

## Example Code Snippets

### Container Launch Command Implementation

```go
// internal/cli/container.go
func ContainerLaunchCmd() *cobra.Command {
    var ephemeral bool
    var project string

    cmd := &cobra.Command{
        Use:   "launch <image> <name>",
        Short: "Launch a new container from an image",
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            image := args[0]
            name := args[1]

            mgr := container.NewManager(name)
            if err := mgr.Launch(image, ephemeral); err != nil {
                return fmt.Errorf("failed to launch container: %w", err)
            }

            fmt.Fprintf(os.Stderr, "Container %s launched from %s\n", name, image)
            return nil
        },
    }

    cmd.Flags().BoolVar(&ephemeral, "ephemeral", false, "Create ephemeral container")
    cmd.Flags().StringVar(&project, "project", "default", "Incus project")

    return cmd
}
```

### Container Exec Command Implementation

```go
func ContainerExecCmd() *cobra.Command {
    var user, group *int
    var envVars []string
    var cwd string
    var capture bool

    cmd := &cobra.Command{
        Use:   "exec <name> -- <command>",
        Short: "Execute a command in a container",
        Args:  cobra.MinimumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            containerName := args[0]

            // Find "--" separator
            var command string
            for i, arg := range args {
                if arg == "--" && i+1 < len(args) {
                    command = strings.Join(args[i+1:], " ")
                    break
                }
            }
            if command == "" {
                return fmt.Errorf("no command specified (use -- before command)")
            }

            // Parse env vars
            env := make(map[string]string)
            for _, e := range envVars {
                parts := strings.SplitN(e, "=", 2)
                if len(parts) == 2 {
                    env[parts[0]] = parts[1]
                }
            }

            mgr := container.NewManager(containerName)
            opts := container.ExecCommandOptions{
                User:    user,
                Group:   group,
                Cwd:     cwd,
                Env:     env,
                Capture: capture,
            }

            output, err := mgr.ExecCommand(command, opts)
            if err != nil {
                return err
            }

            if capture {
                fmt.Println(output)
            }
            return nil
        },
    }

    cmd.Flags().IntVar(&user, "user", 0, "User ID to run as")
    cmd.Flags().IntVar(&group, "group", 0, "Group ID to run as")
    cmd.Flags().StringArrayVar(&envVars, "env", []string{}, "Environment variable (KEY=VALUE)")
    cmd.Flags().StringVar(&cwd, "cwd", "/workspace", "Working directory")
    cmd.Flags().BoolVar(&capture, "capture", false, "Capture output")

    return cmd
}
```

### Image Cleanup Command Implementation

```go
// internal/cli/images.go
func ImageCleanupCmd() *cobra.Command {
    var keepCount int

    cmd := &cobra.Command{
        Use:   "cleanup <prefix>",
        Short: "Delete old image versions, keeping only N most recent",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            prefix := args[0]

            if keepCount <= 0 {
                return fmt.Errorf("--keep must be > 0")
            }

            deleted, kept, err := image.Cleanup(prefix, keepCount)
            if err != nil {
                return err
            }

            fmt.Fprintf(os.Stderr, "Cleanup complete:\n")
            fmt.Fprintf(os.Stderr, "Deleted %d old version(s)\n", len(deleted))
            fmt.Fprintf(os.Stderr, "Kept %d recent version(s)\n", len(kept))

            return nil
        },
    }

    cmd.Flags().IntVar(&keepCount, "keep", 0, "Number of versions to keep (required)")
    cmd.MarkFlagRequired("keep")

    return cmd
}
```

---

## Questions / Clarifications

1. **Incus Group**: Should all commands use the `incus-admin` group via `sg` wrapper, or should this be configurable?

2. **Error Output**: Should errors go to stderr while data goes to stdout (standard practice)?

3. **Verbose Mode**: Should there be a global `--verbose` flag for debugging?

4. **Project Support**: All commands should support `--project` flag for Incus projects?

5. **Container Naming**: Any restrictions or conventions for container names?

6. **Image Alias Format**: Should there be validation for alias formats (especially for versioned images)?

---

## Success Criteria

✅ All container operations can be performed via `coi container` commands
✅ All file operations can be performed via `coi file` commands
✅ All image operations can be performed via `coi image` commands
✅ Custom images can be built via `coi build custom`
✅ Image versioning and cleanup works via `coi image cleanup`
✅ All commands support JSON output for programmatic use
✅ Exit codes are consistent and documented
✅ Full API documentation exists
✅ Integration tests pass
✅ claude_yard can replace all direct Incus calls with `coi` calls
