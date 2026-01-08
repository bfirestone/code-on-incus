"""
Scenario: Resume auto-detects latest session.

Flow:
1. Start session
2. Exit
3. Run --resume (no ID) - should find latest

Expected:
- Auto-detects the session we just created
- Successfully resumes
- Conversation history is preserved
"""

import time

from pexpect import TIMEOUT
from support.helpers import exit_claude, spawn_coi, wait_for_container_ready, wait_for_prompt


def test_resume_auto_detects_latest(coi_binary, cleanup_containers, workspace_dir):
    """Test that --resume without argument auto-detects latest session."""
    # Start session
    child1 = spawn_coi(coi_binary, ["shell"], cwd=workspace_dir, timeout=90)

    wait_for_container_ready(child1, timeout=60)
    wait_for_prompt(child1, timeout=90)

    # Send something memorable
    child1.sendline("The color is PURPLE")
    time.sleep(5)

    # Exit
    exit_claude(child1, timeout=10)
    time.sleep(3)

    # Resume without specifying session ID
    child2 = spawn_coi(coi_binary, ["shell", "--resume"], cwd=workspace_dir, timeout=90)

    wait_for_container_ready(child2, timeout=60)

    # Should auto-detect
    output = child2.logfile_read.get_output()
    assert "Auto-detected session:" in output

    wait_for_prompt(child2, timeout=90)

    # Check memory
    child2.sendline("What color did I mention?")

    try:
        child2.expect(r"PURPLE", timeout=60)
        remembered = True
    except TIMEOUT:
        remembered = False

    exit_claude(child2, timeout=10)

    assert remembered, "Auto-resume did not restore conversation"
