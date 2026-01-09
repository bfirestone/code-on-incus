# Simple Fake Claude Usage

## Best Approach: Install test-claude Alongside Real Claude

### 1. Update build scripts to include test-claude

Add to `internal/image/builder.go` in the sandbox/privileged build:

```bash
# Install test-claude alongside real claude
cp /workspace/testdata/fake-claude/claude /usr/local/bin/test-claude
chmod +x /usr/local/bin/test-claude
```

### 2. Add env var support in shell.go

In `internal/cli/shell.go`, check for test mode:

```go
// Determine which Claude to use
claudeBinary := "claude"
if os.Getenv("COI_USE_TEST_CLAUDE") == "1" {
    claudeBinary = "test-claude"
}

// Use claudeBinary instead of hardcoded "claude"
```

### 3. Usage in Tests

```python
def test_something_fast(coi_binary, workspace_dir):
    """Test using fake Claude (10x faster)."""

    # Set env var to use test-claude
    env = os.environ.copy()
    env["COI_USE_TEST_CLAUDE"] = "1"

    child = spawn_coi(
        coi_binary,
        ["shell"],
        cwd=workspace_dir,
        env=env  # ← Uses test-claude instead of claude!
    )

    # Test proceeds normally, but 10x faster!
    # ...
```

### 4. Benefits

- ✅ **Same image** for all tests (no separate test image)
- ✅ **One env var** to switch (COI_USE_FAKE_CLAUDE=1)
- ✅ **No PATH manipulation** needed
- ✅ **Easy toggle** per-test or per-suite
- ✅ **Both available** in same container

### 5. Running Tests

```bash
# Fast tests with fake Claude
COI_USE_TEST_CLAUDE=1 pytest tests/shell/ephemeral/

# Smoke tests with real Claude
pytest tests/shell/persistent/

# All tests (hybrid)
pytest tests/shell/  # ephemeral=fake, persistent=real
```

## Implementation Plan

1. Update `internal/image/builder.go` to install test-claude
2. Add `COI_USE_FAKE_CLAUDE` support to `internal/cli/shell.go`
3. Update ephemeral tests to set `COI_USE_FAKE_CLAUDE=1`
4. Keep persistent tests using real Claude (no env var)

## Alternative: Even Simpler

Just install test-claude and have tests explicitly use it:

```python
child = spawn_coi(
    coi_binary,
    ["run", "test-claude"],  # Run test-claude directly
    cwd=workspace_dir
)
```

This avoids needing COI_USE_FAKE_CLAUDE support in the code!
