# Fake Claude CLI for Testing

This directory contains a test stub that simulates Claude Code CLI behavior without requiring a license or authentication.

## Purpose

The fake Claude allows tests to:
- Run much faster (no Claude startup time or API calls)
- Work without authentication/licenses
- Have predictable, deterministic behavior
- Test container orchestration logic independently

## How It Works

The `claude` script simulates:
- Initial setup prompts (text style, keyboard shortcuts)
- Session state management
- Resume functionality
- Permission bypass buttons
- Basic interactive chat loop

## Usage in Tests

Use the `fake_claude_path` fixture in tests:

```python
def test_with_fake_claude(coi_binary, fake_claude_path):
    """Test using fake Claude instead of real one."""
    # Build custom image with fake Claude
    result = subprocess.run(
        [coi_binary, "shell", "--image", "test-fake-claude"],
        env={"PATH": f"{fake_claude_path}:{os.environ['PATH']}"},
        ...
    )
```

## Keeping Some Smoke Tests

A few tests should still use the real Claude CLI to ensure integration works:
- `tests/shell/integration/real_claude_smoke.py` - Basic shell start/stop with real Claude
- `tests/shell/integration/real_claude_resume.py` - Resume functionality with real Claude

All other tests can use the fake Claude for speed and reliability.

## Extending

To add more realistic behavior:
1. Parse more Claude CLI flags
2. Simulate specific prompts/responses
3. Add tool use simulation
4. Handle `.claude` directory structure more accurately
