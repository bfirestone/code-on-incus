"""
Scenario: Basic shell session starts successfully.

Verifies:
- Container launches
- Claude starts
- Can exit cleanly
"""

from support.helpers import exit_claude, spawn_coi, wait_for_container_ready, wait_for_prompt


def test_shell_starts_successfully(coi_binary, cleanup_containers, workspace_dir):
    """Test that we can start a basic shell session."""
    print("\n=== Starting shell session ===")
    child = spawn_coi(coi_binary, ["shell"], cwd=workspace_dir)

    print("Waiting for container setup...")
    wait_for_container_ready(child, timeout=60)
    print("✓ Container ready")

    print("Waiting for Claude prompt...")
    wait_for_prompt(child, timeout=90)
    print("✓ Claude ready")

    # Show what we see in the terminal
    output = child.logfile_read.get_output()
    print(f"\n--- Terminal output (last 800 chars) ---\n{output[-800:]}\n--- End output ---\n")

    print("Exiting...")
    exit_success = exit_claude(child, timeout=10)

    print(f"Exit success: {exit_success}")
    print(f"Exit status: {child.exitstatus}")

    assert exit_success, "Failed to exit cleanly"
    assert child.exitstatus == 0 or child.exitstatus is None, (
        f"Unexpected exit status: {child.exitstatus}"
    )
    print("✓ Test passed")
