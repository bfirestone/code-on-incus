"""
Scenario: Resumed session has valid credentials.

This specifically tests the bug where resumed sessions
had no/invalid credentials and started as "unauthorized".

Flow:
1. Start session
2. Exit
3. Resume
4. Verify Claude can make API calls (has credentials)

Expected:
- Session resumes with working credentials
- Claude responds (not unauthorized)
- Can make API calls
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


@pytest.mark.skip(reason="Test the specific issue being debugged")
def test_resume_has_valid_credentials(coi_binary, cleanup_containers, workspace_dir):
    """Test that resumed session has valid credentials."""
    # Start session
    child1 = spawn_coi(coi_binary, ["shell"], cwd=workspace_dir, timeout=90)

    wait_for_container_ready(child1, timeout=60)

    session_id = get_session_id_from_output(child1.logfile_read.get_output())

    wait_for_prompt(child1, timeout=90)
    child1.sendline("test message")
    time.sleep(5)
    exit_claude(child1, timeout=10)
    time.sleep(3)

    # Resume
    child2 = spawn_coi(coi_binary, ["shell", "--resume", session_id], cwd=workspace_dir, timeout=90)

    wait_for_container_ready(child2, timeout=60)
    wait_for_prompt(child2, timeout=90)

    # Try to use Claude - this will fail if no credentials
    child2.sendline("Say OK if you can hear me")

    # Wait for response
    try:
        # If credentials work, Claude will respond
        # If no credentials, we'll get an error or authorization prompt
        child2.expect(r"(OK|hear)", timeout=60)
        has_credentials = True

        # Check we didn't get authorization error
        output = child2.logfile_read.get_output()
        has_auth_error = "unauthorized" in output.lower() or "not authorized" in output.lower()
    except TIMEOUT:
        has_credentials = False
        has_auth_error = True

    exit_claude(child2, timeout=10)

    assert has_credentials, "No response from Claude - likely missing credentials"
    assert not has_auth_error, "Got authorization error - credentials not working"
