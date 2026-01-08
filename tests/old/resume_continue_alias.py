"""
Scenario: --continue works as alias for --resume.

Flow:
1. Start session
2. Exit
3. Use --continue instead of --resume

Expected:
- --continue flag works identically to --resume
- Session is restored
- Conversation history preserved
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


def test_resume_with_continue_alias(coi_binary, cleanup_containers, workspace_dir):
    """Test that --continue works as alias for --resume."""
    # Start session
    child1 = spawn_coi(coi_binary, ["shell"], cwd=workspace_dir, timeout=90)

    wait_for_container_ready(child1, timeout=60)

    session_id = get_session_id_from_output(child1.logfile_read.get_output())
    assert session_id

    wait_for_prompt(child1, timeout=90)
    child1.sendline("The animal is ELEPHANT")
    time.sleep(5)
    exit_claude(child1, timeout=10)
    time.sleep(3)

    # Use --continue instead of --resume
    child2 = spawn_coi(
        coi_binary, ["shell", "--continue", session_id], cwd=workspace_dir, timeout=90
    )

    wait_for_container_ready(child2, timeout=60)
    wait_for_prompt(child2, timeout=90)

    child2.sendline("What animal did I say?")

    try:
        child2.expect(r"ELEPHANT", timeout=60)
        remembered = True
    except TIMEOUT:
        remembered = False

    exit_claude(child2, timeout=10)

    assert remembered, "--continue alias did not work"
