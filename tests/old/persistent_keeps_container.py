"""
Scenario: Persistent flag keeps container running.

Verifies:
- Container persists after exit
- Can reconnect to same container
- Container is reused (not recreated)
"""

import time

from pexpect import TIMEOUT
from support.helpers import (
    exit_claude,
    get_session_id_from_output,
    spawn_coi,
    wait_for_container_ready,
    wait_for_prompt,
)


def test_persistent_flag_keeps_container(coi_binary, cleanup_containers, workspace_dir):
    """Test that --persistent flag keeps the container running."""
    print("\n=== Testing persistent container ===")

    print("Starting persistent session...")
    child = spawn_coi(coi_binary, ["shell", "--persistent"], cwd=workspace_dir)

    print("Waiting for container...")
    wait_for_container_ready(child, timeout=60)
    print("✓ Container ready")

    output = child.logfile_read.get_output()
    session_id = get_session_id_from_output(output)
    print(f"Session ID: {session_id}")
    assert session_id, "Could not extract session ID from output"

    print("Waiting for Claude prompt...")
    wait_for_prompt(child, timeout=90)
    print("✓ Claude ready")

    print("Exiting first session...")
    exit_claude(child, timeout=10)

    print("\nWaiting 2 seconds...")
    time.sleep(2)

    print("Starting second session (should reuse container)...")
    child2 = spawn_coi(coi_binary, ["shell", "--persistent"], cwd=workspace_dir)

    print("Checking if container was reused...")
    try:
        child2.expect(r"(reusing|Restarting)", timeout=30)
        reused = True
        print("✓ Found 'reusing' or 'Restarting' in output")
    except TIMEOUT:
        reused = False
        output2 = child2.logfile_read.get_output()
        print("✗ Did not find reuse indication")
        print(f"\nOutput:\n{output2}")

    exit_claude(child2, timeout=10)

    assert reused, "Expected to reuse persistent container but created new one"
    print("✓ Test passed")
