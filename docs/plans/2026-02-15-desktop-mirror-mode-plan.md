# Desktop Mirror Mode Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wire up existing but unconnected config fields and add workspace path mirroring so COI containers can replicate the host desktop environment exactly.

**Architecture:** Additive changes to 7 existing Go files + 1 new build script. Config fields `code_user`, `code_uid` already exist but are unused by runtime code — we connect them. New fields `mount_tool_config` and `workspace_container_path` are added. All defaults preserve current behavior.

**Tech Stack:** Go, TOML config, Incus containers, Bash (build script), pacman (Arch Linux)

**Design doc:** `docs/plans/2026-02-15-desktop-mirror-mode-design.md`

---

### Task 1: Add new config fields and defaults

**Files:**
- Modify: `internal/config/config.go:62-66` (DefaultsConfig struct)
- Modify: `internal/config/config.go:75-82` (IncusConfig struct)
- Modify: `internal/config/config.go:185-200` (DefaultConfig function)
- Modify: `internal/config/config.go:314-350` (Merge function)
- Test: `internal/config/config_test.go`

**Step 1: Write the failing tests**

Add to `internal/config/config_test.go`:

```go
func TestDesktopMirrorConfigDefaults(t *testing.T) {
	cfg := GetDefaultConfig()

	if cfg.Defaults.MountToolConfig {
		t.Error("Expected default MountToolConfig to be false")
	}

	if cfg.Incus.WorkspaceContainerPath != "/workspace" {
		t.Errorf("Expected default WorkspaceContainerPath '/workspace', got '%s'", cfg.Incus.WorkspaceContainerPath)
	}
}

func TestDesktopMirrorConfigMerge(t *testing.T) {
	base := GetDefaultConfig()

	other := &Config{
		Defaults: DefaultsConfig{
			MountToolConfig: true,
		},
		Incus: IncusConfig{
			WorkspaceContainerPath: "/home/testuser/projects",
			CodeUser:               "testuser",
			CodeUID:                1001,
		},
	}

	base.Merge(other)

	if !base.Defaults.MountToolConfig {
		t.Error("Expected MountToolConfig to be true after merge")
	}

	if base.Incus.WorkspaceContainerPath != "/home/testuser/projects" {
		t.Errorf("Expected WorkspaceContainerPath '/home/testuser/projects', got '%s'", base.Incus.WorkspaceContainerPath)
	}

	if base.Incus.CodeUser != "testuser" {
		t.Errorf("Expected CodeUser 'testuser', got '%s'", base.Incus.CodeUser)
	}

	if base.Incus.CodeUID != 1001 {
		t.Errorf("Expected CodeUID 1001, got %d", base.Incus.CodeUID)
	}
}

func TestDesktopMirrorConfigMerge_EmptyWorkspacePath(t *testing.T) {
	base := GetDefaultConfig()

	other := &Config{
		Incus: IncusConfig{
			WorkspaceContainerPath: "",
		},
	}

	base.Merge(other)

	// Empty string means "mirror host path" — should NOT override default
	// The Merge function should only override when the value is explicitly set
	// Since empty string is the zero value, it should NOT override
	if base.Incus.WorkspaceContainerPath != "/workspace" {
		t.Errorf("Expected WorkspaceContainerPath to remain '/workspace' when other is empty, got '%s'", base.Incus.WorkspaceContainerPath)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run TestDesktopMirror -v`
Expected: FAIL — `MountToolConfig` and `WorkspaceContainerPath` fields don't exist

**Step 3: Add the struct fields**

In `internal/config/config.go`, add `MountToolConfig` to `DefaultsConfig` (after line 65):

```go
type DefaultsConfig struct {
	Image           string `toml:"image"`
	Persistent      bool   `toml:"persistent"`
	Model           string `toml:"model"`
	MountToolConfig bool   `toml:"mount_tool_config"`
}
```

Add `WorkspaceContainerPath` to `IncusConfig` (after line 81):

```go
type IncusConfig struct {
	Project                string `toml:"project"`
	Group                  string `toml:"group"`
	CodeUID                int    `toml:"code_uid"`
	CodeUser               string `toml:"code_user"`
	DisableShift           bool   `toml:"disable_shift"`
	WorkspaceContainerPath string `toml:"workspace_container_path"`
}
```

**Step 4: Set defaults in `GetDefaultConfig`**

In the `Incus` section of `GetDefaultConfig()` (around line 195-200), add the default:

```go
Incus: IncusConfig{
	Project:                "default",
	Group:                  "incus-admin",
	CodeUID:                1000,
	CodeUser:               "code",
	WorkspaceContainerPath: "/workspace",
},
```

`MountToolConfig` defaults to `false` (Go zero value), no explicit setting needed.

**Step 5: Add merge logic**

In the `Merge` method, after the `DisableShift` merge (around line 385-387), add:

```go
if other.Incus.WorkspaceContainerPath != "" {
	c.Incus.WorkspaceContainerPath = other.Incus.WorkspaceContainerPath
}
```

For `MountToolConfig`, add after the `Persistent` merge (around line 325):

```go
c.Defaults.MountToolConfig = other.Defaults.MountToolConfig
```

**Note on empty string semantics:** `workspace_container_path = ""` in TOML means "mirror host path." But in Merge, empty string is the zero value and shouldn't override. Users who want mirror mode set this explicitly. The resolution logic (empty → use host path) happens in `shell.go`, not in Merge. If a user wants to set it explicitly to empty in their config to trigger mirror mode, they'll need to use a sentinel. Actually, the simplest approach: add a special sentinel value. Instead, let's use a different approach — the user sets `workspace_container_path = "mirror"` to trigger mirroring, and we check for that in shell.go. But actually, the design doc says empty = mirror. The problem is TOML merge can't distinguish "not set" from "explicitly empty."

**Revised approach:** Use the string `"mirror"` as the explicit trigger value instead of empty string. This avoids the merge ambiguity. When the user writes `workspace_container_path = "mirror"`, shell.go resolves it to the host path. When not set, it defaults to `"/workspace"`.

Update the test:

```go
func TestDesktopMirrorConfigMerge_MirrorWorkspacePath(t *testing.T) {
	base := GetDefaultConfig()

	other := &Config{
		Incus: IncusConfig{
			WorkspaceContainerPath: "mirror",
		},
	}

	base.Merge(other)

	if base.Incus.WorkspaceContainerPath != "mirror" {
		t.Errorf("Expected WorkspaceContainerPath 'mirror', got '%s'", base.Incus.WorkspaceContainerPath)
	}
}
```

**Step 6: Run tests to verify they pass**

Run: `go test ./internal/config/ -run TestDesktopMirror -v`
Expected: PASS

**Step 7: Run full config test suite**

Run: `go test ./internal/config/ -v`
Expected: All PASS (existing tests unaffected)

**Step 8: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add mount_tool_config and workspace_container_path config fields"
```

---

### Task 2: Wire up `code_user` and `code_uid` in `SetupOptions` and `SetupResult`

**Files:**
- Modify: `internal/session/setup.go:84-100` (SetupOptions struct)
- Modify: `internal/session/setup.go:102-111` (SetupResult struct)
- Modify: `internal/session/setup.go:188-197` (home dir resolution)
- Modify: `internal/session/setup.go:483,512,550,680-681,688-689` (CodeUID references)

**Step 1: Add fields to `SetupOptions` and `SetupResult`**

In `SetupOptions` (around line 98), add:

```go
type SetupOptions struct {
	WorkspacePath          string
	Image                  string
	Persistent             bool
	ResumeFromID           string
	Slot                   int
	MountConfig            *MountConfig
	SessionsDir            string
	CLIConfigPath          string
	Tool                   tool.Tool
	NetworkConfig          *config.NetworkConfig
	DisableShift           bool
	LimitsConfig           *config.LimitsConfig
	IncusProject           string
	ProtectedPaths         []string
	CodeUser               string // Container username (default: "code")
	CodeUID                int    // Container user UID (default: 1000)
	WorkspaceContainerPath string // Mount point for workspace in container (default: "/workspace", "mirror" = use host path)
	MountToolConfig        bool   // Mount tool config dir directly instead of copying
	Logger                 func(string)
}
```

In `SetupResult` (around line 110), add:

```go
type SetupResult struct {
	ContainerName          string
	Manager                *container.Manager
	NetworkManager         *network.Manager
	TimeoutMonitor         *limits.TimeoutMonitor
	HomeDir                string
	RunAsRoot              bool
	Image                  string
	CodeUID                int
	WorkspaceContainerPath string
	MountToolConfig        bool
}
```

**Step 2: Use config values in home dir resolution**

Replace the home dir logic at line 196:

```go
// Before:
result.HomeDir = "/home/" + container.CodeUser

// After:
codeUser := opts.CodeUser
if codeUser == "" {
	codeUser = container.CodeUser
}
result.HomeDir = "/home/" + codeUser
```

**Step 3: Resolve and store workspace container path**

After home dir resolution (around line 197), add:

```go
// Resolve workspace container path
result.WorkspaceContainerPath = opts.WorkspaceContainerPath
if result.WorkspaceContainerPath == "mirror" {
	result.WorkspaceContainerPath = opts.WorkspacePath
}
if result.WorkspaceContainerPath == "" {
	result.WorkspaceContainerPath = "/workspace"
}
```

Store CodeUID and MountToolConfig in result:

```go
result.CodeUID = opts.CodeUID
if result.CodeUID == 0 {
	result.CodeUID = container.CodeUID
}
result.MountToolConfig = opts.MountToolConfig
```

**Step 4: Replace all `container.CodeUID` references in setup.go**

Replace every `container.CodeUID` in setup.go with `result.CodeUID`. There are 6 occurrences at lines 483, 512, 550, 680, 681, 688-689. Each one is in a `Chown` or `chown` call.

Example (line 483):
```go
// Before:
if err := mgr.Chown(statePath, container.CodeUID, container.CodeUID); err != nil {
// After:
if err := mgr.Chown(statePath, result.CodeUID, result.CodeUID); err != nil {
```

And the chown command (line 689):
```go
// Before:
chownCmd := fmt.Sprintf("chown -R %d:%d %s", container.CodeUID, container.CodeUID, stateDir)
// After:
chownCmd := fmt.Sprintf("chown -R %d:%d %s", result.CodeUID, result.CodeUID, stateDir)
```

**Step 5: Replace workspace mount path**

At line 286:
```go
// Before:
if err := result.Manager.MountDisk("workspace", opts.WorkspacePath, "/workspace", useShift, false); err != nil {
// After:
if err := result.Manager.MountDisk("workspace", opts.WorkspacePath, result.WorkspaceContainerPath, useShift, false); err != nil {
```

**Step 6: Run existing tests**

Run: `go test ./internal/session/ -v`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/session/setup.go
git commit -m "feat: wire up code_user, code_uid, and workspace_container_path in session setup"
```

---

### Task 3: Update `security.go` to accept workspace container path

**Files:**
- Modify: `internal/session/security.go:16,37`
- Test: `internal/session/security_test.go`

**Step 1: Add `containerWorkspacePath` parameter to `SetupSecurityMounts`**

```go
// Before (line 16):
func SetupSecurityMounts(mgr *container.Manager, workspacePath string, protectedPaths []string, useShift bool) error {

// After:
func SetupSecurityMounts(mgr *container.Manager, workspacePath string, containerWorkspacePath string, protectedPaths []string, useShift bool) error {
```

Pass it through to `setupProtectedPath`:

```go
// Before (line 22):
if err := setupProtectedPath(mgr, workspacePath, relPath, useShift); err != nil {

// After:
if err := setupProtectedPath(mgr, workspacePath, containerWorkspacePath, relPath, useShift); err != nil {
```

**Step 2: Update `setupProtectedPath` to use the parameter**

```go
// Before (line 35):
func setupProtectedPath(mgr *container.Manager, workspacePath, relPath string, useShift bool) error {

// After:
func setupProtectedPath(mgr *container.Manager, workspacePath, containerWorkspacePath, relPath string, useShift bool) error {
```

```go
// Before (line 37):
containerPath := filepath.Join("/workspace", relPath)

// After:
containerPath := filepath.Join(containerWorkspacePath, relPath)
```

**Step 3: Update callers in `setup.go`**

At line 298 of `setup.go`:
```go
// Before:
if err := SetupSecurityMounts(result.Manager, opts.WorkspacePath, opts.ProtectedPaths, useShift); err != nil {

// After:
if err := SetupSecurityMounts(result.Manager, opts.WorkspacePath, result.WorkspaceContainerPath, opts.ProtectedPaths, useShift); err != nil {
```

**Step 4: Update tests**

Update all `SetupSecurityMounts` calls in `security_test.go` to include the new parameter. For example:

```go
// Before:
err = SetupSecurityMounts(nil, tmpDir, []string{}, false)
// After:
err = SetupSecurityMounts(nil, tmpDir, "/workspace", []string{}, false)
```

Do this for all 4 test calls (lines 79, 84, 98, 124).

**Step 5: Run tests**

Run: `go test ./internal/session/ -run TestSetup -v`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/session/security.go internal/session/security_test.go internal/session/setup.go
git commit -m "feat: pass workspace container path through security mounts"
```

---

### Task 4: Add `mount_tool_config` logic in setup.go

**Files:**
- Modify: `internal/session/setup.go:381-428` (steps 9 and 11)

**Step 1: Guard the resume/restore path (step 9) with `!opts.MountToolConfig`**

At line 383:
```go
// Before:
if opts.ResumeFromID != "" && opts.Tool != nil && opts.Tool.ConfigDirName() != "" {

// After:
if opts.ResumeFromID != "" && opts.Tool != nil && opts.Tool.ConfigDirName() != "" && !opts.MountToolConfig {
```

**Step 2: Add mount-vs-copy logic in step 11**

Replace the step 11 block (lines 404-428) with:

```go
// 11. Setup CLI tool config
if opts.Tool != nil && opts.Tool.ConfigDirName() != "" {
	configDirName := opts.Tool.ConfigDirName()

	if opts.MountToolConfig && opts.CLIConfigPath != "" {
		// Desktop mirror mode: mount host config dir directly
		opts.Logger(fmt.Sprintf("Mounting %s config directory from host...", opts.Tool.Name()))
		containerConfigPath := filepath.Join(result.HomeDir, configDirName)
		if err := result.Manager.MountDisk("tool-config", opts.CLIConfigPath, containerConfigPath, useShift, false); err != nil {
			return nil, fmt.Errorf("failed to mount tool config: %w", err)
		}
		opts.Logger(fmt.Sprintf("Mounted %s -> %s", opts.CLIConfigPath, containerConfigPath))

		// Also mount the state file (e.g., ~/.claude.json)
		stateFile := fmt.Sprintf(".%s.json", opts.Tool.Name())
		hostStateFile := filepath.Join(filepath.Dir(opts.CLIConfigPath), stateFile)
		if _, err := os.Stat(hostStateFile); err == nil {
			containerStateFile := filepath.Join(result.HomeDir, stateFile)
			if err := result.Manager.MountDisk("tool-state", hostStateFile, containerStateFile, useShift, false); err != nil {
				opts.Logger(fmt.Sprintf("Warning: Failed to mount %s: %v", stateFile, err))
			} else {
				opts.Logger(fmt.Sprintf("Mounted %s -> %s", hostStateFile, containerStateFile))
			}
		}
	} else if opts.CLIConfigPath != "" && opts.ResumeFromID == "" {
		// Existing behavior: copy essential files
		if _, err := os.Stat(opts.CLIConfigPath); err == nil {
			if !skipLaunch {
				opts.Logger(fmt.Sprintf("Setting up %s config...", opts.Tool.Name()))
				if err := setupCLIConfig(result.Manager, opts.CLIConfigPath, result.HomeDir, opts.Tool, opts.Logger); err != nil {
					opts.Logger(fmt.Sprintf("Warning: Failed to setup %s config: %v", opts.Tool.Name(), err))
				}
			} else {
				opts.Logger(fmt.Sprintf("Reusing existing %s config (persistent container)", opts.Tool.Name()))
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to check %s config directory: %w", opts.Tool.Name(), err)
		}
	} else if opts.ResumeFromID != "" {
		opts.Logger(fmt.Sprintf("Resuming session - using restored %s config", opts.Tool.Name()))
	}
} else if opts.Tool != nil {
	opts.Logger(fmt.Sprintf("Tool %s uses ENV-based auth, skipping config setup", opts.Tool.Name()))
}
```

**Step 3: Add validation — warn if home paths don't match when mounting**

After the mount logic, add:

```go
if opts.MountToolConfig {
	hostHome, _ := os.UserHomeDir()
	if hostHome != result.HomeDir {
		opts.Logger(fmt.Sprintf(
			"Warning: mount_tool_config is enabled but host home (%s) differs from container home (%s). "+
				"Absolute paths in tool config may not resolve correctly. Consider setting code_user to match your host user.",
			hostHome, result.HomeDir,
		))
	}
}
```

**Step 4: Run tests**

Run: `go test ./internal/session/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/session/setup.go
git commit -m "feat: add mount_tool_config support — mount instead of copy"
```

---

### Task 5: Add user-exists validation in setup.go

**Files:**
- Modify: `internal/session/setup.go` (after container ready check, around line 280)

**Step 1: Add user validation after container is ready**

After the `waitForReady` call and before the workspace mount, add:

```go
// Validate that configured user exists in the container image
if !result.RunAsRoot {
	codeUser := opts.CodeUser
	if codeUser == "" {
		codeUser = container.CodeUser
	}
	checkCmd := fmt.Sprintf("id %s 2>/dev/null", codeUser)
	if _, err := result.Manager.ExecCommand(checkCmd, container.ExecCommandOptions{Capture: true}); err != nil {
		return nil, fmt.Errorf(
			"user '%s' does not exist in image '%s' — build a custom image with this user:\n"+
				"  coi build custom coi-desktop --base images:archlinux --script scripts/build/desktop-arch.sh",
			codeUser, image,
		)
	}
}
```

**Step 2: Run tests**

Run: `go test ./internal/session/ -v`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/session/setup.go
git commit -m "feat: validate configured user exists in container image"
```

---

### Task 6: Update `cleanup.go` to skip save when tool config is mounted

**Files:**
- Modify: `internal/session/cleanup.go:17-27` (CleanupOptions struct)
- Modify: `internal/session/cleanup.go:55-56` (save guard)
- Modify: `internal/session/cleanup.go:133` (home dir)

**Step 1: Add `MountToolConfig` to `CleanupOptions`**

```go
type CleanupOptions struct {
	ContainerName   string
	SessionID       string
	Persistent      bool
	SessionsDir     string
	SaveSession     bool
	Workspace       string
	Tool            tool.Tool
	NetworkManager  *network.Manager
	MountToolConfig bool // When true, skip session save (config is directly mounted)
	HomeDir         string // Container home dir (avoids recomputing)
	Logger          func(string)
}
```

**Step 2: Guard saveSessionData with `!opts.MountToolConfig`**

At line 55:
```go
// Before:
if opts.SaveSession && exists && opts.SessionID != "" && opts.SessionsDir != "" && opts.Tool != nil && opts.Tool.ConfigDirName() != "" {

// After:
if opts.SaveSession && !opts.MountToolConfig && exists && opts.SessionID != "" && opts.SessionsDir != "" && opts.Tool != nil && opts.Tool.ConfigDirName() != "" {
```

**Step 3: Use `opts.HomeDir` in `saveSessionData` instead of hardcoded path**

At line 133:
```go
// Before:
homeDir := "/home/" + container.CodeUser

// After:
homeDir := opts.HomeDir
if homeDir == "" {
	homeDir = "/home/" + container.CodeUser
}
```

Wait — `saveSessionData` is a standalone function, not a method on `CleanupOptions`. It receives individual parameters. The cleanest approach: pass `homeDir` as a parameter.

Update the function signature:

```go
// Before:
func saveSessionData(mgr *container.Manager, sessionID string, persistent bool, workspace string, sessionsDir string, t tool.Tool, logger func(string)) error {

// After:
func saveSessionData(mgr *container.Manager, sessionID string, persistent bool, workspace string, sessionsDir string, homeDir string, t tool.Tool, logger func(string)) error {
```

Replace line 133:
```go
// Before:
homeDir := "/home/" + container.CodeUser

// After (use parameter, with fallback):
if homeDir == "" {
	homeDir = "/home/" + container.CodeUser
}
```

Update the call site at line 56:
```go
// Before:
if err := saveSessionData(mgr, opts.SessionID, opts.Persistent, opts.Workspace, opts.SessionsDir, opts.Tool, opts.Logger); err != nil {

// After:
if err := saveSessionData(mgr, opts.SessionID, opts.Persistent, opts.Workspace, opts.SessionsDir, opts.HomeDir, opts.Tool, opts.Logger); err != nil {
```

**Step 4: Run tests**

Run: `go test ./internal/session/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/session/cleanup.go
git commit -m "feat: skip session save when tool config is mounted, use configurable home dir"
```

---

### Task 7: Update `cli/shell.go` — populate options and use result values

**Files:**
- Modify: `internal/cli/shell.go:196-210` (SetupOptions population)
- Modify: `internal/cli/shell.go:245-254` (CleanupOptions population)
- Modify: `internal/cli/shell.go:399,428,479` (CodeUID references)
- Modify: `internal/cli/shell.go:560,575,636,643,658` ("/workspace" references)

**Step 1: Resolve config values and populate SetupOptions**

Before the `setupOpts` construction (around line 195), add resolution logic:

```go
// Resolve code user/uid from config with fallback to constants
codeUser := cfg.Incus.CodeUser
if codeUser == "" {
	codeUser = container.CodeUser
}
codeUID := cfg.Incus.CodeUID
if codeUID == 0 {
	codeUID = container.CodeUID
}

// Resolve workspace container path
workspaceContainerPath := cfg.Incus.WorkspaceContainerPath
if workspaceContainerPath == "mirror" {
	workspaceContainerPath = absWorkspace
}
```

Add new fields to the `setupOpts` struct literal:

```go
setupOpts := session.SetupOptions{
	WorkspacePath:          absWorkspace,
	Image:                  imageName,
	Persistent:             persistent,
	ResumeFromID:           resumeID,
	Slot:                   slotNum,
	SessionsDir:            sessionsDir,
	CLIConfigPath:          cliConfigPath,
	Tool:                   toolInstance,
	NetworkConfig:          &networkConfig,
	DisableShift:           cfg.Incus.DisableShift,
	LimitsConfig:           limitsConfig,
	IncusProject:           cfg.Incus.Project,
	ProtectedPaths:         protectedPaths,
	CodeUser:               codeUser,
	CodeUID:                codeUID,
	WorkspaceContainerPath: workspaceContainerPath,
	MountToolConfig:        cfg.Defaults.MountToolConfig,
}
```

**Step 2: Update CleanupOptions**

At line 245:

```go
cleanupOpts := session.CleanupOptions{
	ContainerName:   result.ContainerName,
	SessionID:       sessionID,
	Persistent:      persistent,
	SessionsDir:     sessionsDir,
	SaveSession:     true,
	Workspace:       absWorkspace,
	Tool:            toolInstance,
	NetworkManager:  result.NetworkManager,
	MountToolConfig: result.MountToolConfig,
	HomeDir:         result.HomeDir,
}
```

**Step 3: Replace `container.CodeUID` with `result.CodeUID`**

At line 399:
```go
// Before:
user := container.CodeUID
// After:
user := result.CodeUID
```

At line 479:
```go
// Before:
user := container.CodeUID
// After:
user := result.CodeUID
```

**Step 4: Replace all `"/workspace"` with `result.WorkspaceContainerPath`**

There are 8 occurrences. Replace each:

Line 428: `Cwd: "/workspace"` → `Cwd: result.WorkspaceContainerPath`
Line 560: `Cwd: "/workspace"` → `Cwd: result.WorkspaceContainerPath`
Line 575: `-c /workspace` → `-c %s` with `result.WorkspaceContainerPath` as format arg
Line 636: `-c /workspace` → `-c %s` with `result.WorkspaceContainerPath` as format arg
Line 643: `Cwd: "/workspace"` → `Cwd: result.WorkspaceContainerPath`
Line 658: `Cwd: "/workspace"` → `Cwd: result.WorkspaceContainerPath`

For the tmux commands (lines 575, 636), they're `fmt.Sprintf` calls. Change from:

```go
"tmux new-session -d -s %s -c /workspace \"bash -c 'trap : INT; %s %s; exec bash'\"",
tmuxSessionName, envExports, cliCmd,
```

To:

```go
"tmux new-session -d -s %s -c %s \"bash -c 'trap : INT; %s %s; exec bash'\"",
tmuxSessionName, result.WorkspaceContainerPath, envExports, cliCmd,
```

**Step 5: Build to verify compilation**

Run: `go build ./...`
Expected: Compiles without errors

**Step 6: Commit**

```bash
git add internal/cli/shell.go
git commit -m "feat: use configured code_user, code_uid, and workspace path in shell command"
```

---

### Task 8: Update `cli/attach.go` and `cli/run.go`

**Files:**
- Modify: `internal/cli/attach.go:132,135,170,173`
- Modify: `internal/cli/run.go:180,235-236`

**Step 1: Update `attach.go`**

`attach.go` uses `container.CodeUID` and `"/workspace"` directly. It doesn't go through `session.Setup()`, so it needs to resolve from config independently.

At the top of the attach command function, add resolution:

```go
codeUID := cfg.Incus.CodeUID
if codeUID == 0 {
	codeUID = container.CodeUID
}
workspaceContainerPath := cfg.Incus.WorkspaceContainerPath
if workspaceContainerPath == "mirror" {
	workspaceContainerPath = absWorkspace
}
if workspaceContainerPath == "" {
	workspaceContainerPath = "/workspace"
}
```

Then replace the 4 references:
- Line 132: `container.CodeUID` → `codeUID`
- Line 135: `"/workspace"` → `workspaceContainerPath`
- Line 170: `container.CodeUID` → `codeUID`
- Line 173: `"/workspace"` → `workspaceContainerPath`

**Step 2: Update `run.go`**

Same pattern. Add resolution at the top:

```go
codeUID := cfg.Incus.CodeUID
if codeUID == 0 {
	codeUID = container.CodeUID
}
workspaceContainerPath := cfg.Incus.WorkspaceContainerPath
if workspaceContainerPath == "mirror" {
	workspaceContainerPath = absWorkspace
}
if workspaceContainerPath == "" {
	workspaceContainerPath = "/workspace"
}
```

Replace references:
- Line 180: `"/workspace"` → `workspaceContainerPath`
- Line 235: `container.CodeUID` → `codeUID`
- Line 236: `container.CodeUID` → `codeUID` and `"/workspace"` → `workspaceContainerPath`

**Step 3: Build to verify**

Run: `go build ./...`
Expected: Compiles

**Step 4: Commit**

```bash
git add internal/cli/attach.go internal/cli/run.go
git commit -m "feat: use configured code_uid and workspace path in attach and run commands"
```

---

### Task 9: Create the Arch-based desktop build script

**Files:**
- Create: `scripts/build/desktop-arch.sh`

**Step 1: Write the build script**

Create `scripts/build/desktop-arch.sh` with the full content from the design document (Section 5). The script:
- Starts from `images:archlinux`
- Installs packages via `pacman` (curl, wget, git, jq, tmux, base-devel, nodejs, docker, github-cli, python, etc.)
- Creates user `bfirestone` with UID 1000, wheel+docker groups, passwordless sudo
- Installs Claude CLI via the native installer
- Configures power management wrappers
- Installs dummy test stub if present (optional for custom builds)

Use the exact script content from the design doc.

**Step 2: Make it executable**

Run: `chmod +x scripts/build/desktop-arch.sh`

**Step 3: Verify the script is syntactically valid**

Run: `bash -n scripts/build/desktop-arch.sh`
Expected: No errors

**Step 4: Commit**

```bash
git add scripts/build/desktop-arch.sh
git commit -m "feat: add Arch Linux desktop build script for mirror mode"
```

---

### Task 10: Build and test the desktop image

**Step 1: Build the base `coi` image (if not already built)**

Run: `coi build`
Expected: Builds successfully (or skips if already exists)

**Step 2: Build the desktop image**

Run: `coi build custom coi-desktop --base images:archlinux --script scripts/build/desktop-arch.sh --force`
Expected: Builds successfully. Watch for:
- DNS resolution working
- pacman packages installing
- User `bfirestone` being created
- Claude CLI installing

**Step 3: Verify the image exists**

Run: `incus image list --format csv -c l | grep coi-desktop`
Expected: Shows `coi-desktop` alias

---

### Task 11: Create config and end-to-end test

**Step 1: Create a test config file**

Create `~/.config/coi/config.toml` (or update it) with:

```toml
[defaults]
image = "coi-desktop"
persistent = true
mount_tool_config = true

[incus]
code_user = "bfirestone"
code_uid = 1000
workspace_container_path = "mirror"

[mounts]
default = [
    { host = "~/.config/tmux", container = "/home/bfirestone/.config/tmux" },
    { host = "~/.tmux", container = "/home/bfirestone/.tmux" },
]
```

**Step 2: Build the Go binary**

Run: `go build -o coi ./cmd/coi/`
Expected: Compiles

**Step 3: Test with `coi shell`**

Run from a test directory: `./coi shell -w ~/devspace/personal/github/code-on-incus`

Verify:
- Container starts with user `bfirestone` (run `whoami` in the shell)
- Workspace is mounted at `/home/bfirestone/devspace/personal/github/code-on-incus` (run `pwd`)
- `~/.claude` is mounted (check `ls -la ~/.claude/settings.json`)
- `~/.claude.json` is accessible (check `ls -la ~/.claude.json`)
- tmux uses your Catppuccin theme and Ctrl+Space prefix
- Claude starts with your statusline, plugins, and MCP servers

**Step 4: Test backwards compatibility**

Temporarily rename/remove the config changes and verify default behavior still works:
- `code` user
- `/workspace` mount
- Config files copied (not mounted)

**Step 5: Commit final test results and any fixes**

```bash
git add -A
git commit -m "feat: desktop mirror mode — end-to-end verified"
```

---

### Task 12: Run full test suite

**Step 1: Run all unit tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 2: Run vet and lint**

Run: `go vet ./...`
Expected: No issues

**Step 3: Fix any issues found**

If any tests fail, fix and re-run.

**Step 4: Final commit**

```bash
git add -A
git commit -m "test: ensure all tests pass with desktop mirror mode changes"
```
