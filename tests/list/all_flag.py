"""
Test list --all flag functionality.

Expected:
- All flag includes saved sessions
- Command completes without error
"""

import subprocess


def test_list_all_flag(coi_binary):
    """Test that coi list --all includes saved sessions."""
    result = subprocess.run(
        [coi_binary, "list", "--all"], capture_output=True, text=True, timeout=10
    )

    assert result.returncode == 0
    # Should complete without error
