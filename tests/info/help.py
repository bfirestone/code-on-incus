"""
Test info command help functionality.

Expected:
- Help flag works
- Help mentions session ID parameter
"""

import subprocess


def test_info_help_flag(coi_binary):
    """Test that coi info --help shows help."""
    result = subprocess.run(
        [coi_binary, "info", "--help"], capture_output=True, text=True, timeout=5
    )

    assert result.returncode == 0
    assert "info" in result.stdout.lower()
    assert "usage:" in result.stdout.lower() or "session" in result.stdout.lower()


def test_info_help_mentions_session_id(coi_binary):
    """Test that info help mentions session ID parameter."""
    result = subprocess.run(
        [coi_binary, "info", "--help"], capture_output=True, text=True, timeout=5
    )

    assert result.returncode == 0
    output = result.stdout.lower()

    # Help should mention session
    assert "session" in output, "Help should mention session parameter"
