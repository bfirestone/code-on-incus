"""
Test completion error handling for invalid shell.

Expected:
- Invalid shell shows appropriate error
"""

import subprocess


def test_completion_invalid_shell_shows_error(coi_binary):
    """Test that invalid shell shows appropriate error."""
    result = subprocess.run(
        [coi_binary, "completion", "invalid-shell-xyz"],
        capture_output=True,
        text=True,
        timeout=5,
    )

    # Should show error
    assert result.returncode != 0, "Invalid shell should return non-zero exit code"

    output = result.stdout + result.stderr
    # Should mention error or valid options
    assert "error" in output.lower() or "invalid" in output.lower() or "usage" in output.lower(), (
        "Should indicate invalid shell"
    )
