# Internal API Refactoring - Claude-Agnostic Naming

## Summary
Renamed internal functions, variables, and struct fields to use CLI-agnostic terminology instead of Claude-specific names. This is a **non-breaking change** - all external APIs (CLI commands, flags, config files) remain unchanged.

## Changes Made

### Functions Renamed (3)
- `runClaude()` → `runCLI()` 
- `runClaudeInTmux()` → `runCLIInTmux()`
- `GetClaudeSessionID()` → `GetCLISessionID()`
- `setupClaudeConfig()` → `setupCLIConfig()`

### Variables Renamed
- `claudeBinary` → `cliBinary`
- `claudeCmd` → `cliCmd`
- `claudeSessionID` → `cliSessionID`
- `hostClaudeConfigPath` → `hostCLIConfigPath`
- `hostClaudePath` → `hostCLIConfigPath`

### Struct Fields Renamed (1)
- `SetupOptions.ClaudeConfigPath` → `SetupOptions.CLIConfigPath`

### Comments Updated
- Updated function and parameter documentation to use "CLI tool" instead of "Claude"
- Updated inline comments referencing "Claude's session ID" → "CLI's session ID"

## Files Modified (3)
1. `internal/cli/shell.go` - Main CLI execution logic
2. `internal/session/setup.go` - Session setup and config management  
3. `internal/session/cleanup.go` - Session cleanup and ID extraction

## Impact
- ✅ **Zero breaking changes** - All changes are internal
- ✅ **Compiles successfully** - Code verified to build
- ✅ **CLI works** - Tested `coi --help` and `coi version`
- ✅ **Semantically clearer** - Code now uses generic terminology
- ✅ **Foundation for future** - Enables easy addition of config-based tool selection

## What Stays Claude-Specific (Intentional)
- `.claude` directory path (still hardcoded, will be configurable in future)
- Default binary name: `"claude"` (will become configurable)
- User-facing help text (will be addressed separately)
- Config file paths like `.claude-on-incus.toml` (backwards compatibility)

## Next Steps
This refactoring enables:
1. Adding `cli_binary` config option (e.g., `cli_binary = "aider"`)
2. Adding `state_dir` config option (e.g., `state_dir = "~/.aider"`)
3. Per-tool profiles in config

## Testing
- [x] Code compiles
- [x] Binary runs
- [ ] Integration tests (run separately with `COI_USE_DUMMY=1`)
