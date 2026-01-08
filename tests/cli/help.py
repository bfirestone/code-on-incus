"""
Test CLI help commands and basic startup.

Flow:
1. Test --help flag shows help text
2. Test -h shorthand works
3. Test subcommands have help
4. Test --version flag works
5. Verify no crashes on basic commands

Expected:
- CLI starts without errors
- Help text is displayed correctly
- Exit codes are 0 for help commands
- No containers are created
"""

import subprocess


def test_main_help_flag(coi_binary):
    """Test that coi --help displays help text."""
    result = subprocess.run([coi_binary, "--help"], capture_output=True, text=True, timeout=5)

    assert result.returncode == 0, f"Expected exit code 0, got {result.returncode}"
    assert "claude-on-incus" in result.stdout.lower()
    assert "usage:" in result.stdout.lower() or "examples:" in result.stdout.lower()


def test_main_help_shorthand(coi_binary):
    """Test that coi -h works as shorthand for --help."""
    result = subprocess.run([coi_binary, "-h"], capture_output=True, text=True, timeout=5)

    assert result.returncode == 0
    assert "claude-on-incus" in result.stdout.lower()


def test_version_flag(coi_binary):
    """Test that coi --version displays version."""
    result = subprocess.run([coi_binary, "--version"], capture_output=True, text=True, timeout=5)

    assert result.returncode == 0
    # Version should be in stdout or stderr
    output = result.stdout + result.stderr
    assert "version" in output.lower() or "coi" in output.lower()


def test_shell_help(coi_binary):
    """Test that coi shell --help works."""
    result = subprocess.run(
        [coi_binary, "shell", "--help"], capture_output=True, text=True, timeout=5
    )

    assert result.returncode == 0
    assert "shell" in result.stdout.lower()
    assert "usage:" in result.stdout.lower()


def test_list_help(coi_binary):
    """Test that coi list --help works."""
    result = subprocess.run(
        [coi_binary, "list", "--help"], capture_output=True, text=True, timeout=5
    )

    assert result.returncode == 0
    assert "list" in result.stdout.lower()


def test_attach_help(coi_binary):
    """Test that coi attach --help works."""
    result = subprocess.run(
        [coi_binary, "attach", "--help"], capture_output=True, text=True, timeout=5
    )

    assert result.returncode == 0
    assert "attach" in result.stdout.lower()


def test_tmux_help(coi_binary):
    """Test that coi tmux --help works."""
    result = subprocess.run(
        [coi_binary, "tmux", "--help"], capture_output=True, text=True, timeout=5
    )

    assert result.returncode == 0
    assert "tmux" in result.stdout.lower()


def test_build_help(coi_binary):
    """Test that coi build --help works."""
    result = subprocess.run(
        [coi_binary, "build", "--help"], capture_output=True, text=True, timeout=5
    )

    assert result.returncode == 0
    assert "build" in result.stdout.lower()


def test_images_help(coi_binary):
    """Test that coi images --help works."""
    result = subprocess.run(
        [coi_binary, "images", "--help"], capture_output=True, text=True, timeout=5
    )

    assert result.returncode == 0
    assert "images" in result.stdout.lower()


def test_invalid_command_exits_nonzero(coi_binary):
    """Test that invalid commands exit with non-zero code."""
    result = subprocess.run(
        [coi_binary, "nonexistent-command"],
        capture_output=True,
        text=True,
        timeout=5,
    )

    assert result.returncode != 0, "Invalid command should exit with non-zero code"


def test_help_shows_common_commands(coi_binary):
    """Test that help text mentions common commands."""
    result = subprocess.run([coi_binary, "--help"], capture_output=True, text=True, timeout=5)

    assert result.returncode == 0

    # Check for common commands in help
    common_commands = ["shell", "list", "attach", "build", "images"]
    output = result.stdout.lower()

    found_commands = [cmd for cmd in common_commands if cmd in output]
    assert len(found_commands) >= 3, (
        f"Expected at least 3 common commands in help, found: {found_commands}"
    )


def test_help_without_incus_access(coi_binary):
    """Test that help commands work even without Incus daemon access."""
    # Run without sg - help should still work
    result = subprocess.run(
        [coi_binary, "--help"],
        capture_output=True,
        text=True,
        timeout=5,
        env={"PATH": "/usr/bin:/bin"},  # Minimal environment
    )

    assert result.returncode == 0
    assert "claude-on-incus" in result.stdout.lower()


def test_shell_help_shows_flags(coi_binary):
    """Test that shell --help shows important flags."""
    result = subprocess.run(
        [coi_binary, "shell", "--help"], capture_output=True, text=True, timeout=5
    )

    assert result.returncode == 0
    output = result.stdout.lower()

    # Check for important flags
    important_flags = ["--slot", "--persistent", "--privileged", "--tmux"]
    found_flags = [flag for flag in important_flags if flag in output]

    assert len(found_flags) >= 3, f"Expected important flags in help, found: {found_flags}"
