# Incomplete Test Analysis - claude-on-incus

**Analysis Date:** 2026-01-09
**Total Tests:** 102
**Incomplete/Stub Tests:** 36 tests (35% of suite)
**Properly Implemented E2E Tests:** 66 tests (65% of suite)

---

## Executive Summary

Out of 102 tests, **36 tests (35%) are incomplete or don't provide real end-to-end testing**:

- **20 tests** only check help text without testing actual functionality
- **16 tests** have weak assertions or incomplete coverage
- **0 tests** have TODO comments (good!)

The incomplete tests fall into these categories:
1. Help-text-only tests that never invoke actual commands
2. Tests with minimal assertions (only checking exit code 0)
3. Tests using `--help` flag to avoid testing the actual feature
4. Tests with weak keyword matching instead of proper validation

---

## Category 1: Help-Text-Only Tests (20 tests)

These tests **ONLY verify help output** without testing any actual functionality. They should be converted to full E2E tests or removed if redundant.

### 1.1 Main Help Tests

#### `tests/main_help_flag.py::test_main_help_flag`
**Current:** Checks if `coi --help` shows "claude-on-incus" and "usage:"
**Problem:** No functional testing
**Should Do:**
- Keep help test minimal
- Add separate test that actually runs a command (e.g., `coi version`)
- Verify binary execution works

#### `tests/main_help_shorthand.py::test_main_help_shorthand`
**Current:** Tests `coi -h` shows help
**Problem:** Duplicate of above; no functionality
**Should Do:**
- Merge with main_help_flag test
- Add test that verifies `-h` equals `--help` output

---

### 1.2 Subcommand Help-Only Tests

#### `tests/shell/test_shell_help.py::test_shell_help`
**Current:** `coi shell --help` shows "shell" in output
**Problem:** No actual shell launch testing
**Should Do:**
- **Full E2E Test:** Launch ephemeral shell, wait for Claude prompt, send simple prompt ("echo hello"), verify response, exit cleanly
- Verify container created and cleaned up
- Test both with and without tmux

#### `tests/shell/test_shell_help_flags.py::test_shell_help_shows_flags`
**Current:** Verifies help shows `--slot`, `--persistent`, `--privileged`, `--tmux`
**Problem:** Never tests what these flags actually do
**Should Do:**
- **Test `--slot`:** Launch two shells on different slots, verify both containers exist
- **Test `--persistent`:** Launch persistent shell, exit, verify container still exists (stopped)
- **Test `--privileged`:** Launch privileged shell, verify it can use Docker/Git/SSH
- **Test `--tmux`:** Launch with tmux, verify tmux session created, can attach/detach

#### `tests/attach/test_attach_help.py::test_attach_help`
**Current:** `coi attach --help` shows help
**Problem:** No attach functionality testing
**Should Do:**
- **Full E2E Test:** Launch persistent shell in background, use `coi attach` to attach to it, send command, verify response, detach
- Test attach with specific slot: `coi attach --slot 1`
- Test attach when multiple containers running
- Test attach failure when no containers

#### `tests/build_cmd/test_build_help.py::test_build_help`
**Current:** `coi build --help` shows help
**Problem:** No build functionality
**Should Do:**
- **Full E2E Test:** Build sandbox image, verify it exists, launch container from it, verify Claude works
- Build privileged image, verify Git/Docker/SSH available
- Test force rebuild with `--force`

#### `tests/tmux/test_tmux_help.py::test_tmux_help`
**Current:** `coi tmux --help` shows help
**Problem:** No tmux functionality
**Should Do:**
- **Full E2E Test:** Launch Claude in tmux, verify tmux session exists
- Detach from tmux (Ctrl+B D), verify container still running
- Re-attach with `coi tmux attach`, verify session restored
- Test tmux list command

#### `tests/images/images_help.py::test_images_help`
**Current:** `coi images --help` shows help
**Problem:** Already have better tests in `images/` - this is redundant
**Should Do:**
- **Remove this test** - covered by `image/list_with_json.py` and other image tests

#### `tests/list/list_help.py::test_list_help`
**Current:** `coi list --help` shows help
**Problem:** No list functionality
**Should Do:**
- **Full E2E Test:** Launch persistent container, run `coi list`, verify container appears in output
- Verify output format (columns, alignment)
- Test `coi list --all` shows stopped containers too
- Launch ephemeral container, verify it appears in list while running

#### `tests/help/help_command_basic.py::test_help_command`
**Current:** `coi help` shows help
**Problem:** Redundant with main_help_flag
**Should Do:**
- **Merge with main help test** or remove
- Add test that `coi help shell` equals `coi shell --help`

#### `tests/help/help_command_with_subcommand.py::test_help_command_with_subcommand`
**Current:** `coi help shell` shows help
**Problem:** No functionality; redundant
**Should Do:**
- **Remove** - covered by test_shell_help

#### `tests/help/help_shows_commands.py::test_help_shows_common_commands`
**Current:** Help mentions shell, list, attach, build, images
**Problem:** Just parses help text
**Should Do:**
- Keep minimal for documentation verification
- Add separate tests that actually run each command

#### `tests/info/help_flag.py::test_info_help_flag`
**Current:** `coi info --help` shows help
**Problem:** No info functionality
**Should Do:**
- **Full E2E Test:** Launch persistent shell, create session, exit, run `coi info <session-id>`, verify output shows session details
- Verify info shows: session ID, container name, workspace, creation time, last used time
- Test info with nonexistent session (error handling)

#### `tests/info/help_mentions_session_id.py::test_info_help_mentions_session_id`
**Current:** Help mentions "session"
**Problem:** Just text parsing
**Should Do:**
- **Remove** - redundant with help_flag test

#### `tests/images/help.py::test_images_help_flag`
**Current:** `coi images --help` shows help
**Problem:** Duplicate of images_help.py
**Should Do:**
- **Remove** - duplicate test

#### `tests/completion/help_flag.py::test_completion_help_flag`
**Current:** `coi completion --help` shows help
**Problem:** No completion testing
**Should Do:**
- Keep help test minimal
- Already have good tests in `completion/bash_generation.py`, etc.

#### `tests/completion/help_shows_shells.py::test_completion_help_shows_shells`
**Current:** Help mentions bash, zsh, fish, powershell
**Problem:** Just text parsing
**Should Do:**
- **Remove** - redundant with help_flag test

#### `tests/clean/help_flag.py::test_clean_help_flag`
**Current:** `coi clean --help` shows help
**Problem:** No clean functionality
**Should Do:**
- Keep minimal help test
- Already have good tests in `clean/removes_stopped.py`, etc.

#### `tests/clean/help_shows_options.py::test_clean_help_shows_options`
**Current:** Help mentions container, session, or all
**Problem:** Just text parsing
**Should Do:**
- **Remove** - redundant with help_flag test

#### `tests/completion/no_args.py::test_completion_without_shell_shows_help`
**Current:** `coi completion` without args shows help
**Problem:** Just verifies error handling
**Should Do:**
- Keep minimal (verifies proper error message)
- Already have good completion generation tests

---

## Category 2: Weak Functional Tests (16 tests)

These tests have some functionality but with minimal assertions or incomplete coverage.

### 2.1 Minimal Assertion Tests

#### `tests/images/basic_execution.py::test_images_command_basic`
**Current:** Only checks exit code 0
**Problem:** No output validation
**Should Do:**
- **Verify output format:** Check for column headers (ALIAS, SIZE, CREATED)
- Build a test image, verify it appears in list
- Verify COI images section shows coi-sandbox, coi-privileged status
- Test `--all` flag shows more images

#### `tests/images/output_verification.py::test_images_shows_output`
**Current:** Only verifies output is not empty
**Problem:** No content validation
**Should Do:**
- **Verify output structure:** Check for "COI Images:", "Remote Images:", "Custom Images:" sections
- Verify each built COI image shows checkmark (✓) or X (✗)
- Verify remote images section mentions ubuntu/debian/alpine

#### `tests/images/coi_images_mentioned.py::test_images_mentions_coi_images`
**Current:** Checks for vague keywords: "coi", "image", "no", "available"
**Problem:** Overly broad keyword matching
**Should Do:**
- **Specific validation:** Check for exact strings "coi-sandbox", "coi-privileged"
- If not built, verify shows "not built" message
- If built, verify shows checkmark

#### `tests/list/basic_execution.py::test_list_command_basic`
**Current:** Only checks exit code 0
**Problem:** No list validation
**Should Do:**
- **Full E2E Test:**
  1. Clean up all test containers first
  2. Run `coi list` with no containers, verify "no running containers" message
  3. Launch persistent container
  4. Run `coi list`, verify container appears with correct name, slot, status
  5. Exit container (stop it)
  6. Run `coi list`, verify no running containers
  7. Run `coi list --all`, verify stopped container appears

#### `tests/list/all_flag.py::test_list_all_flag`
**Current:** Only checks exit code 0
**Problem:** No validation of `--all` behavior
**Should Do:**
- **Full E2E Test:**
  1. Launch persistent container, exit (stopped state)
  2. Run `coi list` without `--all`, verify stopped container NOT shown
  3. Run `coi list --all`, verify stopped container IS shown
  4. Compare outputs to ensure `--all` includes more data

#### `tests/list/empty_output.py::test_list_shows_appropriate_message_when_empty`
**Current:** Only checks output is not None (meaningless)
**Problem:** Barely a test
**Should Do:**
- **Proper validation:**
  1. Clean up all test containers
  2. Run `coi list`
  3. Verify output contains "No running containers" or similar message
  4. Verify exit code 0 (not an error condition)

#### `tests/list/without_permissions.py::test_list_without_incus_permissions`
**Current:** Only checks exit code is not None (always true!)
**Problem:** Meaningless assertion
**Should Do:**
- **Proper permission test:**
  1. Run `coi list` without `sg incus-admin` wrapper (if possible)
  2. Verify non-zero exit code
  3. Verify error message mentions permissions or "incus-admin group"
- **OR Remove** if not testable in current environment

#### `tests/info/missing_session_error.py::test_info_without_session_id`
**Current:** Checks for vague keywords: "session", "usage", "required", "error"
**Problem:** Weak validation
**Should Do:**
- **Specific error validation:**
  1. Run `coi info` without session ID
  2. Verify non-zero exit code (usage error, exit code 2)
  3. Verify stderr contains specific message like "session ID required" or shows usage
  4. Verify no crash, graceful error

#### `tests/info/nonexistent_session_error.py::test_info_with_nonexistent_session`
**Current:** Checks for vague keywords: "not found", "error", "does not exist"
**Problem:** Weak validation
**Should Do:**
- **Specific error validation:**
  1. Run `coi info 00000000-0000-0000-0000-000000000000` (fake UUID)
  2. Verify exit code 1 (not found error)
  3. Verify error message contains the fake UUID and "not found"
  4. Test with invalid UUID format, verify proper error message

---

### 2.2 Incomplete Feature Tests

#### `tests/attach/basic_attach.py::test_attach_no_containers`
**Current:** Tests attach fails when no containers running
**Problem:** Only tests error case; no actual attach functionality
**Should Do:**
- **Full E2E Test:**
  1. Launch persistent shell on slot 1 (don't interact)
  2. In separate process, run `coi attach --slot 1`
  3. Send prompt via attached session
  4. Verify response appears
  5. Detach (Ctrl+C or exit)
  6. Verify container still running
  7. Clean up container

#### `tests/attach/basic_attach.py::test_attach_to_specific_slot`
**Current:** Runs `coi attach --slot 5 --help` (cheating with --help!)
**Problem:** Uses `--help` to avoid testing actual feature
**Should Do:**
- **Full E2E Test:**
  1. Launch two persistent shells: slot 1 and slot 2
  2. Attach to slot 1, send prompt, verify correct container responds
  3. Attach to slot 2, send prompt, verify correct container responds
  4. Test attach to non-existent slot 99, verify error
  5. Clean up both containers

#### `tests/clean/without_args.py::test_clean_without_args_shows_help_or_runs`
**Current:** Only checks output exists; comment says "behavior depends on implementation"
**Problem:** Test is uncertain about expected behavior
**Should Do:**
- **Determine correct behavior first, then test:**
  - **If default is to show help:** Verify help text shown
  - **If default is to clean containers:** Test that stopped containers are removed
  - **Recommendation:** `coi clean` without args should require confirmation or show usage

#### `tests/clean/mentions_targets.py::test_clean_mentions_containers_or_sessions`
**Current:** Checks for vague keywords without testing actual behavior
**Problem:** Help text validation instead of functionality
**Should Do:**
- **Full E2E Test:**
  1. Launch and stop a container
  2. Create a session directory
  3. Run `coi clean` (with auto-confirm), verify both cleaned
  4. Test `coi clean --containers`, verify only containers cleaned
  5. Test `coi clean --sessions`, verify only sessions cleaned

#### `tests/clean/all_flag.py::test_clean_all_flag`
**Current:** Uses `coi clean --all --help` to avoid testing the dangerous feature
**Problem:** Explicitly avoids testing the actual flag
**Should Do:**
- **Safe E2E Test:**
  1. Create test containers with `coi-test-` prefix (isolated from user containers)
  2. Launch and stop 3 test containers
  3. Run `coi clean --all --force` (force to skip confirmation)
  4. Verify all 3 test containers removed
  5. Verify user containers (without coi-test- prefix) NOT touched

#### `tests/version/version_flag_synonym.py::test_version_flag_synonym`
**Current:** Runs `coi -v` but comment says "We don't assert success here"
**Problem:** Incomplete test
**Should Do:**
- **Complete test:**
  1. Run `coi -v`
  2. Verify exit code 0
  3. Verify output contains version number (regex: `\d+\.\d+\.\d+`)
  4. Verify `coi -v` output equals `coi version` output

---

## Category 3: Tests That Need Better Assertions

These tests work but have weak validation that should be strengthened:

#### `tests/completion/bash_generation.py`
**Current:** Checks if bash completion generated
**Enhancement Needed:**
- Verify completion script has proper bash syntax
- Test that it includes all commands (shell, run, build, etc.)
- Verify flag completion works (basic syntax check)

#### `tests/completion/zsh_generation.py`
**Current:** Checks if zsh completion generated
**Enhancement Needed:**
- Verify zsh-specific syntax
- Check for command and flag completions

#### `tests/completion/fish_generation.py`
**Current:** Checks if fish completion generated
**Enhancement Needed:**
- Verify fish-specific syntax
- Check for command completions

---

## Tests With Good E2E Implementation (Keep These!)

These tests are properly implemented with full end-to-end validation:

✅ **Container Operations:**
- `container/launch_ephemeral.py` - Full lifecycle with verification
- `container/launch_persistent.py` - Persistence validation
- `container/start_stop_delete.py` - Full state machine testing
- `container/exec_with_capture.py` - Output validation
- `container/mount_disk.py` - Mount verification

✅ **File Operations:**
- `file/push_single.py` - File transfer with content verification
- `file/push_directory.py` - Directory sync with validation
- `file/pull_directory.py` - Pull with content check
- `file/error_without_recursive.py` - Error handling

✅ **Image Operations:**
- `image/list_with_json.py` - JSON format validation
- `image/list_with_prefix.py` - Filter testing
- `image/publish_and_delete.py` - Full lifecycle
- `image/exists_check.py` - Boolean validation
- `image/cleanup_versions.py` - Versioning logic

✅ **Build Operations:**
- `build/simple.py` - Full custom build with verification
- `build/with_base.py` - Base image testing
- `build/script_not_found.py` - Error handling
- `build/force_rebuild.py` - Force logic

✅ **Shell Operations:**
- `shell/persistent/container_persists.py` - Full persistence test
- `shell/persistent/container_reused.py` - Reuse validation
- `shell/persistent/filesystem_persistence.py` - File persistence
- `shell/ephemeral/with_tmux/resume_basic.py` - Full interactive session
- `shell/ephemeral/with_tmux/file_persistence.py` - File isolation
- `shell/persistent/test_slot_uniqueness.py` - Slot isolation

✅ **Clean Operations:**
- `clean/removes_stopped.py` - Actual cleanup with verification
- `clean/preserves_running.py` - Safety check
- `clean/confirmation_prompt.py` - Interaction testing
- `clean/force_flag.py` - Skip confirmation

✅ **Other Commands:**
- `run/basic_execution.py` - Full command exec with output
- `kill/single_container.py` - Force kill with verification
- `sudo/sudo_works.py` - Permission testing with validation
- `version/version_flag.py` - Version format validation
- `version/version_is_brief.py` - Output format check

---

## Priority Recommendations

### P0 - Critical (Must Fix)

**High-value tests that should be full E2E:**

1. **`shell/test_shell_help.py`** → Convert to full shell launch test
2. **`attach/test_attach_help.py`** → Convert to full attach test
3. **`attach/basic_attach.py`** → Complete both test functions
4. **`list/basic_execution.py`** → Add full list validation
5. **`list/all_flag.py`** → Test `--all` behavior properly
6. **`info/help_flag.py`** → Convert to full info command test
7. **`images/basic_execution.py`** → Add output format validation
8. **`build_cmd/test_build_help.py`** → Convert to sandbox/privileged build test

**These 8 tests cover core functionality and should be proper E2E tests.**

---

### P1 - High (Should Fix)

**Tests that need better assertions:**

9. **`list/empty_output.py`** → Proper message validation
10. **`info/missing_session_error.py`** → Specific error validation
11. **`info/nonexistent_session_error.py`** → Specific error validation
12. **`images/output_verification.py`** → Structure validation
13. **`images/coi_images_mentioned.py`** → Specific string checks
14. **`clean/without_args.py`** → Determine and test correct behavior
15. **`clean/mentions_targets.py`** → Full clean target test
16. **`clean/all_flag.py`** → Safe `--all` flag test
17. **`version/version_flag_synonym.py`** → Complete assertions

**These 9 tests need assertion improvements.**

---

### P2 - Medium (Nice to Have)

**Redundant help tests that can be merged or removed:**

18-20. **Help test cleanup:** Merge/remove duplicate help tests
21-24. **Shell flag tests:** Convert `test_shell_help_flags.py` to 4 separate E2E tests (one per flag)
25-27. **Completion enhancements:** Add syntax validation to completion tests
28. **Tmux help test:** Convert to full tmux workflow test

**These 10 tests are lower priority cleanup.**

---

### P3 - Low (Optional)

**Help tests that can stay minimal:**
- Main help tests (just verify help text exists)
- Completion help tests (just verify error handling)
- Permission tests that can't be easily run in test env

---

## Implementation Strategy

### Phase 1: Critical E2E Conversions (P0)
**Estimated effort:** 8-12 hours
**Impact:** High - core functionality covered

Convert 8 help-only tests to full E2E tests:
1. Shell launch test
2. Attach test
3. List validation test
4. Info command test
5. Images output test
6. Build sandbox/privileged test

### Phase 2: Assertion Improvements (P1)
**Estimated effort:** 4-6 hours
**Impact:** Medium - better test reliability

Strengthen assertions in 9 existing tests:
- Add specific error message checks
- Add output format validation
- Add proper state verification

### Phase 3: Test Cleanup (P2)
**Estimated effort:** 2-4 hours
**Impact:** Low - code quality improvement

Merge/remove duplicate tests, clean up test organization

---

## Summary Statistics

| Category | Count | Percentage |
|----------|-------|------------|
| **Total Tests** | **102** | **100%** |
| Properly Implemented E2E | 66 | 65% |
| Help-Text-Only | 20 | 20% |
| Weak/Incomplete | 16 | 15% |

### After Fixes (Target):
| Category | Count | Percentage |
|----------|-------|------------|
| **Total Tests** | **~90** | **100%** |
| Properly Implemented E2E | ~80 | 89% |
| Minimal Help Tests (OK) | ~10 | 11% |
| Weak/Incomplete | 0 | 0% |

**Goal:** Remove ~12 redundant tests, convert 8 tests to full E2E, strengthen 9 tests
**Result:** 90 high-quality tests with 89% full E2E coverage

---

## Recommendations for Test Quality

### What Makes a Good E2E Test:

1. **Full Workflow:** Create → Use → Verify → Cleanup
2. **State Verification:** Check actual state changes, not just exit codes
3. **Output Validation:** Parse and verify output format and content
4. **Error Cases:** Test both success and failure paths
5. **Cleanup:** Always clean up test resources
6. **Isolation:** Use test-specific prefixes/names

### Examples of Excellent E2E Tests in Your Suite:

- `file/push_single.py` - Creates file, pushes to container, execs command to verify, checks content
- `shell/persistent/container_reused.py` - Launches, verifies container exists, relaunches, verifies same container reused
- `clean/removes_stopped.py` - Creates stopped containers, runs clean, verifies they're gone
- `image/cleanup_versions.py` - Creates multiple versioned images, runs cleanup, verifies correct ones kept/deleted

### Anti-Patterns to Avoid:

❌ Only checking exit code 0 without output validation
❌ Using `--help` flag to avoid testing the actual feature
❌ Vague keyword matching instead of specific validation
❌ Tests with comments like "behavior depends on implementation"
❌ Assertions that always pass (e.g., `assert output is not None`)

---

## Next Steps

1. **Review P0 tests** - Decide which ones to convert to full E2E
2. **Start with `shell/test_shell_help.py`** - Good template for converting help test to E2E
3. **Use existing E2E tests as templates** - Copy patterns from `file/push_single.py`, etc.
4. **Run tests frequently** - Verify E2E tests work reliably
5. **Document test requirements** - Update test docstrings with expected behavior

Your test infrastructure (pexpect, fixtures, helpers) is excellent - you just need to replace the stub tests with real implementations! 🚀
