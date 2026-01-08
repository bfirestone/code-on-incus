"""
Test info command help flag.

Expected:
- Help flag works and shows usage
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
