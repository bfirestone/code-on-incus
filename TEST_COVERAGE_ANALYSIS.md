# Test Coverage Analysis for claude-on-incus

**Analysis Date:** 2026-01-09
**Total Test Functions:** 82
**Test Framework:** pytest with pexpect for interactive testing

---

## Executive Summary

The test suite is **well-structured** with good coverage of new CLI commands (container, file, image) and basic shell functionality. However, there are **significant gaps** in testing core commands (run, attach, tmux, kill), error conditions, security features, and advanced scenarios like concurrent operations and configuration management.

**Coverage Level:** ~60% (good foundation, needs expansion)

---

## Test Structure Overview

### Current Test Organization

```
tests/
├── cli/              # Basic CLI tests (help, version, new commands)
├── clean/            # Clean command tests (9 tests)
├── completion/       # Shell completion tests (7 tests)
├── images/           # Legacy images command tests (4 tests)
├── info/             # Info command tests (4 tests)
├── list/             # List command tests (5 tests)
├── shell/
│   ├── ephemeral/
│   │   ├── without_tmux/  # 3 tests
│   │   └── with_tmux/     # 5 tests
│   └── persistent/        # 8 tests
└── support/          # Test helpers and fixtures
```

### Test Infrastructure Quality

**✅ Strengths:**
- Excellent helper utilities (`helpers.py`) with terminal emulation
- LiveScreenMonitor for async screen updates
- Proper cleanup fixtures to prevent container leaks
- Workspace isolation with UUID-based temp directories
- Smart use of pexpect for interactive testing
- Container prefix separation (`coi-test-`) to avoid user session conflicts

**⚠️ Potential Issues:**
- No parallel test execution safety (tests could conflict if run in parallel)
- Long timeouts (30-90 seconds) could slow down test runs
- Heavy reliance on sleep() for synchronization (race conditions possible)

---

## Coverage Analysis by Command

### 1. `coi` (root command)
**Coverage:** ✅ Good (11 tests)
- Help flags and output
- Version information
- Invalid command handling
- Binary execution without Incus

**Missing:**
- Default behavior (runs shell when no command specified)
- Global flag inheritance to subcommands
- Configuration file loading and hierarchy
- Profile application

---

### 2. `coi shell` (Interactive Claude session)
**Coverage:** ✅ Good (18 tests)
- Basic start/stop with/without tmux
- File persistence in ephemeral mode
- Container persistence in persistent mode
- Basic resume functionality
- Slot uniqueness validation
- Mount/no-mount Claude config

**Missing:**
- **Privileged mode tests** - No tests for `--privileged` flag
- **Environment variable passing** - `--env` flag not tested
- **Storage mounting** - `--storage` flag not tested
- **Image selection** - `--image` flag not tested
- **Profile usage** - `--profile` flag not tested
- **Resume with specific session ID** - Only auto-detect tested
- **Continue flag** - `--continue` alias not tested
- **Workspace flag** - `--workspace` not explicitly tested
- **Multiple slot interaction** - Only uniqueness meta-test exists
- **Session corruption handling** - What if .claude dir is malformed?
- **Container name collision** - What if container already exists?
- **Network issues** - Container startup failures
- **Claude CLI errors** - What if Claude crashes?

---

### 3. `coi run` (Run command in container)
**Coverage:** ❌ None (0 tests)

**Needed:**
- Basic command execution
- With/without tmux
- Persistent vs ephemeral
- Exit code propagation
- Stdout/stderr capture
- Command with arguments
- Long-running commands
- Command failures
- Environment variables
- Working directory
- Resume functionality with run

---

### 4. `coi attach` (Attach to running session)
**Coverage:** ⚠️ Minimal (1 help test only)

**Needed:**
- Attach to running container
- Attach to specific slot
- Attach when no containers running
- Attach when multiple containers
- Detach and re-attach
- Attach to tmux vs non-tmux
- Permission to attach

---

### 5. `coi tmux` (Tmux session management)
**Coverage:** ⚠️ Minimal (1 help test only)

**Needed:**
- Start Claude in tmux
- Attach to tmux session
- Detach from tmux
- List tmux sessions
- Kill tmux session
- Tmux with persistent containers
- Multiple tmux sessions (different slots)
- Tmux session recovery after disconnect

---

### 6. `coi build` (Build images)
**Coverage:** ⚠️ Partial (5 tests for custom, 0 for sandbox/privileged)

**Tested:**
- Custom image with simple script ✅
- Custom image with explicit base ✅
- Custom image force rebuild ✅
- Script not found error ✅
- Custom image verification ✅

**Missing:**
- **Build sandbox image** - Core image not tested
- **Build privileged image** - Core image not tested
- **Force rebuild** - `--force` flag for sandbox/privileged
- **Build failures** - Script errors, network issues
- **Image already exists** - Skip behavior
- **Concurrent builds** - Multiple builds running
- **Build script permissions** - Non-executable scripts
- **Build script output** - Logging and error messages
- **Build with different bases** - Ubuntu versions
- **Build time verification** - Reasonable build duration

---

### 7. `coi images` (Legacy command)
**Coverage:** ✅ Good (4 tests)
- Basic execution ✅
- COI images mentioned ✅
- Help flag ✅
- Output verification ✅

---

### 8. `coi image` (Image management)
**Coverage:** ✅ Good (5 tests)
- List with JSON format ✅
- List with prefix filter ✅
- Publish and delete ✅
- Exists check ✅
- Cleanup old versions ✅

**Missing:**
- **List all images** - `--all` flag behavior
- **List format variations** - Table vs JSON consistency
- **Publish failures** - Container not stopped, permission issues
- **Delete nonexistent image** - Error handling
- **Delete in-use image** - Image being used by container
- **Cleanup edge cases** - Zero images, one image, all same timestamp
- **Cleanup sorting** - Verify correct timestamp extraction
- **Image alias validation** - Invalid characters, length limits

---

### 9. `coi container` (Low-level container ops)
**Coverage:** ✅ Good (4 tests)
- Launch ephemeral ✅
- Launch persistent ✅
- Start/stop/delete lifecycle ✅
- Exec with capture ✅
- Mount disk ✅

**Missing:**
- **Exists/running checks** - Return codes only tested implicitly
- **Exec with user/group** - `--user` and `--group` flags
- **Exec with env vars** - `--env` flag
- **Exec with cwd** - `--cwd` flag
- **Exec failures** - Command not found, permission denied
- **Mount failures** - Source doesn't exist, permission issues
- **Launch from remote image** - images:ubuntu/22.04
- **Launch failures** - Image not found, name conflict
- **Stop force behavior** - Difference between --force and without
- **Delete force behavior** - Delete running container

---

### 10. `coi file` (File operations)
**Coverage:** ✅ Good (4 tests)
- Push single file ✅
- Push directory ✅
- Pull directory ✅
- Error without -r flag ✅

**Missing:**
- **Pull single file** - Not just directories
- **Push/pull large files** - Performance and reliability
- **Push/pull permissions** - File ownership after transfer
- **Push/pull symlinks** - How are symlinks handled?
- **Push/pull special files** - Sockets, devices, etc.
- **Path resolution** - Relative vs absolute paths
- **Container not running** - Can we push/pull to stopped container?
- **Container doesn't exist** - Error handling
- **Source doesn't exist** - Error handling
- **Destination path issues** - Parent dir doesn't exist, permission denied

---

### 11. `coi list` (List containers/sessions)
**Coverage:** ✅ Good (5 tests)
- Basic execution ✅
- All flag ✅
- Empty output ✅
- Help ✅
- Without permissions ✅

**Missing:**
- **List active containers only** - Default behavior
- **List sessions only** - Session data
- **List format** - Output structure validation
- **List with multiple containers** - Proper formatting
- **List with different states** - Running, stopped, ephemeral

---

### 12. `coi info` (Session info)
**Coverage:** ✅ Good (4 tests)
- Help flag ✅
- Help mentions session ID ✅
- Missing session error ✅
- Nonexistent session error ✅

**Missing:**
- **Actual info retrieval** - Get info for valid session
- **Info output format** - What fields are shown
- **Info for different session types** - Ephemeral vs persistent

---

### 13. `coi clean` (Cleanup)
**Coverage:** ✅ Excellent (9 tests)
- All flag ✅
- Confirmation prompt ✅
- Force flag ✅
- Help ✅
- Preserves running containers ✅
- Removes stopped containers ✅
- Sessions flag ✅
- Without args ✅
- Mentions targets ✅

**Missing:**
- **Clean with multiple stopped containers** - Batch cleanup
- **Clean with session data** - Does it clean sessions too?
- **Clean failures** - Permission issues, Incus errors

---

### 14. `coi kill` (Force kill containers)
**Coverage:** ❌ None (0 tests)

**Needed:**
- Kill single container
- Kill multiple containers
- Kill all containers
- Kill with --force flag
- Kill nonexistent container
- Kill already stopped container
- Kill with confirmation prompt
- Kill running Claude session

---

### 15. `coi version` (Version info)
**Coverage:** ✅ Excellent (6 tests)
- Version flag ✅
- Version command ✅
- Version synonym ✅
- Brief output ✅
- Number format ✅
- Without Incus ✅

---

### 16. `coi completion` (Shell completions)
**Coverage:** ✅ Excellent (7 tests)
- Bash generation ✅
- Zsh generation ✅
- Fish generation ✅
- Help flag ✅
- Invalid shell error ✅
- No args ✅
- Help shows shells ✅

---

## Missing Test Categories

### Security & Isolation Tests
**Coverage:** ❌ None

**Needed:**
- **UID/GID mapping verification** - Files created in container have correct ownership on host
- **Credential isolation** - Verify host credentials not exposed to Claude
- **Network isolation** - Container network access
- **File system isolation** - Container can't access host files outside mounts
- **Escape attempts** - Try to break out of container
- **Sudo/privilege escalation** - Verify unprivileged containers

### Configuration & Environment Tests
**Coverage:** ❌ None

**Needed:**
- **Config file loading** - `~/.config/coi/config.toml`
- **Config hierarchy** - System → User → Project → CLI flags
- **Profile application** - `--profile` flag
- **Environment variables** - `COI_*` env vars
- **Invalid config handling** - Malformed TOML
- **Missing config** - Defaults work correctly

### Multi-Session & Concurrency Tests
**Coverage:** ❌ None

**Needed:**
- **Parallel sessions same workspace** - Different slots work
- **Parallel sessions different workspaces** - No conflicts
- **Concurrent builds** - Multiple image builds
- **Concurrent file transfers** - Race conditions
- **Session ID collision** - UUID generation safety
- **Slot allocation** - Auto-slot (0) works correctly

### Error Handling & Edge Cases
**Coverage:** ⚠️ Minimal (scattered across tests)

**Needed:**
- **Incus not available** - Graceful error messages
- **Incus daemon not running** - Recovery suggestions
- **Permission denied** - Not in incus-admin group
- **Disk full** - During image build, file transfer
- **Network timeout** - Image download failures
- **Container startup failures** - Various Incus errors
- **Resource limits** - Too many containers, memory limits
- **Corrupted session data** - Malformed .claude directory
- **Interrupted operations** - SIGTERM during build/transfer

### Performance & Stress Tests
**Coverage:** ❌ None

**Needed:**
- **Large file transfers** - GB-sized files
- **Many containers** - 50+ containers running
- **Long-running sessions** - Hours/days
- **Rapid create/destroy** - Stress test container lifecycle
- **Memory leaks** - Long-running processes
- **Build time consistency** - Reproducible builds

### Integration & E2E Workflows
**Coverage:** ⚠️ Minimal (only basic shell workflows)

**Needed:**
- **Complete workflow: shell → work → resume → work → clean**
- **Complete workflow: build custom → launch → exec → publish → cleanup**
- **Complete workflow: run → attach → detach → kill**
- **Multi-user simulation** - Multiple users on same machine
- **Upgrade scenarios** - Migrate from old version
- **Backup/restore** - Session data portability

### Documentation Accuracy Tests
**Coverage:** ❌ None

**Needed:**
- **README examples work** - All code examples in README are valid
- **Help text accuracy** - Help matches actual behavior
- **API documentation** - Programmatic usage examples work
- **Error messages** - Helpful and actionable

---

## Priority Recommendations

### P0 - Critical (Must Have)
1. **`coi run` command tests** - Core functionality not tested at all
2. **`coi kill` command tests** - Force termination not tested
3. **`coi attach` command tests** - Only help test exists
4. **Security/isolation tests** - UID mapping, credential isolation
5. **Error handling tests** - Incus unavailable, permissions, disk full
6. **Build sandbox/privileged tests** - Core images not tested

### P1 - High (Should Have)
1. **Multi-session/concurrency tests** - Parallel operations
2. **Configuration tests** - Config files, profiles, env vars
3. **`coi tmux` command tests** - Tmux integration not tested
4. **Shell command flag coverage** - Privileged, env, storage, image, profile
5. **Resume with session ID** - Explicit session resume not tested
6. **Container lifecycle edge cases** - Name conflicts, failures, cleanup

### P2 - Medium (Nice to Have)
1. **Performance tests** - Large files, many containers
2. **E2E workflow tests** - Complete user journeys
3. **File operation edge cases** - Symlinks, permissions, large files
4. **Image management edge cases** - Cleanup corner cases, concurrent builds
5. **Documentation accuracy tests** - Ensure examples work

### P3 - Low (Can Defer)
1. **Stress tests** - Extreme load scenarios
2. **Long-running tests** - Multi-day sessions
3. **Upgrade scenarios** - Migration testing
4. **Multi-user simulation** - Rare edge cases

---

## Test Infrastructure Improvements

### Recommended Enhancements

1. **Parallel test execution safety**
   - Add test markers for safe-to-run-parallel vs must-run-serial
   - Use per-test workspace isolation (already implemented)
   - Add container cleanup verification between tests

2. **Reduce reliance on sleep()**
   - Use polling with timeouts instead of fixed delays
   - Implement smart waiting (check conditions every 100ms)
   - Add exponential backoff for retries

3. **Better timeout management**
   - Short timeouts for fast operations (exists, running)
   - Medium timeouts for lifecycle operations (launch, stop)
   - Long timeouts for builds and Claude startup
   - Configurable via environment variable

4. **Structured test output**
   - Add pytest markers for categories (security, performance, etc.)
   - Generate coverage reports by category
   - Track test duration trends

5. **Test data management**
   - Pre-built test images to speed up tests
   - Cached dependencies for builds
   - Fixture for common container states

6. **Error injection framework**
   - Mock Incus failures
   - Simulate disk full, network timeout
   - Test recovery mechanisms

---

## Metrics

### Current Coverage Breakdown

| Category | Tests | Coverage |
|----------|-------|----------|
| CLI Basics | 27 | ✅ Excellent |
| Shell Command | 18 | ✅ Good |
| Container Ops | 4 | ⚠️ Partial |
| File Ops | 4 | ⚠️ Partial |
| Image Ops | 9 | ✅ Good |
| Clean/List/Info | 18 | ✅ Good |
| Completion | 7 | ✅ Excellent |
| **Run Command** | **0** | ❌ None |
| **Attach Command** | **1** | ❌ None |
| **Tmux Command** | **1** | ❌ None |
| **Kill Command** | **0** | ❌ None |
| **Security Tests** | **0** | ❌ None |
| **Config Tests** | **0** | ❌ None |
| **Concurrency** | **0** | ❌ None |
| **Performance** | **0** | ❌ None |

### Estimated Test Additions Needed

- **P0 Critical:** ~35 new tests
- **P1 High:** ~40 new tests
- **P2 Medium:** ~30 new tests
- **P3 Low:** ~20 new tests

**Total:** ~125 additional tests needed for comprehensive coverage
**New Total:** ~207 tests (current 82 + new 125)

---

## Conclusion

The test suite has a **solid foundation** with excellent infrastructure (pexpect, terminal emulation, cleanup fixtures) and good coverage of new commands (container, file, image). However, there are **critical gaps** in testing:

1. Core commands completely untested (run, kill, attach fully, tmux fully)
2. No security/isolation verification
3. No configuration management tests
4. No concurrency/parallel operation tests
5. Limited error handling coverage

**Recommendation:** Focus on **P0 and P1** items (~75 new tests) to achieve production-ready test coverage. This would increase the test count from 82 to ~157, providing solid confidence in the system's reliability and correctness.
