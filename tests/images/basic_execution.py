"""
Test images command basic execution.

Expected:
- Command runs without errors
- Exit code is 0
"""

import subprocess


def test_images_command_basic(coi_binary):
    """Test that coi images runs without errors."""
    result = subprocess.run([coi_binary, "images"], capture_output=True, text=True, timeout=10)

    assert result.returncode == 0, f"Expected exit code 0, got {result.returncode}"
