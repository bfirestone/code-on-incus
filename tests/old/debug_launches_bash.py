"""
Scenario: Debug flag launches bash instead of Claude.

Verifies:
- Gets bash prompt instead of Claude
- Can execute basic commands
- Shell works correctly
"""

from pexpect import EOF
from support.helpers import spawn_coi, wait_for_container_ready


def test_debug_flag_launches_bash(coi_binary, cleanup_containers, workspace_dir):
    """Test that --debug flag launches bash instead of Claude."""
    print("\n=== Testing debug mode (bash) ===")

    print("Starting with --debug flag...")
    child = spawn_coi(coi_binary, ["shell", "--debug"], cwd=workspace_dir, timeout=60)

    print("Waiting for container...")
    wait_for_container_ready(child, timeout=60)
    print("✓ Container ready")

    print("Looking for bash prompt...")
    child.expect(r"[$#]", timeout=30)
    print("✓ Got bash prompt")

    print("Testing echo command...")
    child.sendline("echo 'test'")
    child.expect(r"test", timeout=5)
    print("✓ Echo worked")

    print("Exiting bash...")
    child.sendline("exit")
    child.expect(EOF, timeout=10)
    print("✓ Test passed")
