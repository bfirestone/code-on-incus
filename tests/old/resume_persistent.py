"""
Scenario: Resume a persistent session.

Flow:
1. Start persistent session
2. Exit (container kept running)
3. Resume - should reconnect to same container

Expected:
- Reconnects to existing container
- Session state preserved
- Can continue conversation
- Claude remembers previous context
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


def test_resume_persistent_session(coi_binary, cleanup_containers, workspace_dir):
    """Test resuming a persistent session."""
    print("\n=== Testing persistent resume ===")

    print("Starting persistent session...")
    child1 = spawn_coi(coi_binary, ["shell", "--persistent"], cwd=workspace_dir, timeout=90)

    print("Waiting for container...")
    wait_for_container_ready(child1, timeout=60)
    print("✓ Container ready")

    output = child1.logfile_read.get_output()
    session_id = get_session_id_from_output(output)
    print(f"Session ID: {session_id}")
    assert session_id, "Could not extract session ID"

    print("Waiting for Claude prompt...")
    wait_for_prompt(child1, timeout=90)
    print("✓ Claude ready")

    print("\nSending memorable message: 'the magic number is 42'")
    child1.sendline("Remember this: the magic number is 42")
    time.sleep(5)

    print("Exiting first session...")
    exit_claude(child1, timeout=10)
    time.sleep(3)

    print(f"\nResuming with session ID {session_id}...")
    child2 = spawn_coi(
        coi_binary, ["shell", "--resume", session_id, "--persistent"], cwd=workspace_dir, timeout=90
    )

    print("Waiting for container...")
    wait_for_container_ready(child2, timeout=60)
    print("✓ Container ready")

    output2 = child2.logfile_read.get_output()
    print(f"\nResume output snippet:\n{output2[-300:]}")

    has_resume_mode = "Resume mode:" in output2 or "Resuming session:" in output2
    print(f"Has resume indication: {has_resume_mode}")
    assert has_resume_mode, "Should indicate resume mode"

    print("Waiting for Claude prompt...")
    wait_for_prompt(child2, timeout=90)
    print("✓ Claude ready")

    print("\nAsking: 'What was the magic number?'")
    child2.sendline("What was the magic number?")

    print("Waiting for '42' in response...")
    try:
        child2.expect(r"42", timeout=60)
        remembered = True
        print("✓ Found '42' - Claude remembered!")
    except TIMEOUT:
        remembered = False
        output_final = child2.logfile_read.get_output()
        print("✗ Did not find '42'")
        print(f"\nFull output:\n{output_final}")

    print("Exiting...")
    exit_claude(child2, timeout=10)

    assert remembered, (
        "Claude did not remember the previous conversation - persistent resume failed"
    )
    print("✓ Test passed")
