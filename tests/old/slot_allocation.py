"""
Scenario: Automatic slot allocation for parallel sessions.

Verifies:
- First session gets slot 1
- Second parallel session gets slot 2
- Both can run simultaneously
"""

from support.helpers import exit_claude, spawn_coi, wait_for_container_ready, wait_for_prompt


def test_slot_allocation(coi_binary, cleanup_containers, workspace_dir):
    """Test automatic slot allocation for parallel sessions."""
    print("\n=== Testing slot allocation ===")

    print("Starting first session...")
    child1 = spawn_coi(coi_binary, ["shell", "--persistent"], cwd=workspace_dir, timeout=60)

    print("Waiting for first container...")
    wait_for_container_ready(child1, timeout=60)
    print("✓ First container ready")

    output1 = child1.logfile_read.get_output()
    print(f"\nFirst session output snippet:\n{output1[-500:]}")

    has_slot_1 = "slot 1" in output1.lower() or "Auto-allocated slot 1" in output1
    print(f"First session got slot 1: {has_slot_1}")
    assert has_slot_1, "First session should get slot 1"

    print("Waiting for first Claude prompt...")
    wait_for_prompt(child1, timeout=90)
    print("✓ First Claude ready")

    print("\nStarting second session (parallel)...")
    child2 = spawn_coi(coi_binary, ["shell", "--persistent"], cwd=workspace_dir, timeout=60)

    print("Waiting for second container...")
    wait_for_container_ready(child2, timeout=60)
    print("✓ Second container ready")

    output2 = child2.logfile_read.get_output()
    print(f"\nSecond session output snippet:\n{output2[-500:]}")

    # Should get slot 2 (or higher, since slot 1 is occupied)
    has_different_slot = any(
        f"slot {i}" in output2.lower() or f"using slot {i}" in output2 for i in range(2, 10)
    )
    print(f"Second session got different slot (2+): {has_different_slot}")

    # Cleanup
    print("\nCleaning up...")
    exit_claude(child1, timeout=10)
    exit_claude(child2, timeout=10)

    assert has_different_slot, f"Expected different slot allocation but got: {output2}"
    print("✓ Test passed")
