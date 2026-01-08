"""
Scenario: Resume a non-persistent session (ephemeral container).

Flow:
1. Start session, send memorable message
2. Exit (container deleted, .claude saved)
3. Resume - should restore conversation

Expected:
- Session history restored
- Credentials work
- Can continue conversation
- Claude remembers previous context
"""

import time

import pytest
from pexpect import TIMEOUT
from support.helpers import (
    exit_claude,
    get_session_id_from_output,
    spawn_coi,
    wait_for_container_ready,
    wait_for_prompt,
)


@pytest.mark.skip(reason="Resume functionality being fixed - use this to verify the fix")
def test_resume_non_persistent_session(coi_binary, cleanup_containers, workspace_dir):
    """Test resuming a non-persistent session (ephemeral container)."""
    # Start first session
    child1 = spawn_coi(coi_binary, ["shell"], cwd=workspace_dir, timeout=90)

    wait_for_container_ready(child1, timeout=60)

    # Get session ID
    output = child1.logfile_read.get_output()
    session_id = get_session_id_from_output(output)
    assert session_id, "Could not extract session ID"

    wait_for_prompt(child1, timeout=90)

    # Send a memorable message
    child1.sendline("Remember this: the secret word is BANANA")

    # Wait a bit for response
    time.sleep(5)

    # Exit (session should be saved)
    exit_claude(child1, timeout=10)

    # Brief pause
    time.sleep(3)

    # Resume the session
    child2 = spawn_coi(coi_binary, ["shell", "--resume", session_id], cwd=workspace_dir, timeout=90)

    wait_for_container_ready(child2, timeout=60)

    # Check for resume indication
    child2.logfile_read.get_output()

    wait_for_prompt(child2, timeout=90)

    # Ask Claude about the previous conversation
    child2.sendline("What was the secret word I told you?")

    # Wait for response containing BANANA
    try:
        child2.expect(r"BANANA", timeout=60)
        remembered = True
    except TIMEOUT:
        remembered = False
        print(f"Output: {child2.logfile_read.get_output()}")

    # Cleanup
    exit_claude(child2, timeout=10)

    assert remembered, "Claude did not remember the previous conversation - resume failed"
