"""
Tests for coi attach command - basic attachment functionality.

Tests:
1. Attach fails when no containers running
2. Attach to specific slot works end-to-end
"""

import os
import subprocess
import time

from support.helpers import (
    assert_clean_exit,
    calculate_container_name,
    exit_claude,
    get_container_list,
    send_prompt,
    spawn_coi,
    wait_for_container_deletion,
    wait_for_container_ready,
    wait_for_prompt,
    wait_for_text_in_monitor,
    with_live_screen,
)


def test_attach_no_containers(coi_binary, cleanup_containers):
    """Test attach when no containers are running."""
    result = subprocess.run(
        [coi_binary, "attach"],
        capture_output=True,
        text=True,
        timeout=5,
    )
    # Should exit with success but show message about no containers
    # (exit code 0 is acceptable - it's informational, not an error)
    output = result.stdout + result.stderr
    assert "no" in output.lower(), "Should mention no containers/sessions"


def test_attach_to_specific_slot(coi_binary, cleanup_containers, workspace_dir, fake_claude_path):
    """Test that attach --slot works to connect to a specific container slot."""
    env = os.environ.copy()
    env["PATH"] = f"{fake_claude_path}:{env.get('PATH', '')}"

    # Launch container on slot 3 with tmux
    child1 = spawn_coi(
        coi_binary,
        ["shell", "--persistent", "--slot=3"],
        cwd=workspace_dir,
        env=env
    )

    wait_for_container_ready(child1, timeout=60)
    wait_for_prompt(child1, timeout=90)

    # Interact to verify it works
    with with_live_screen(child1) as monitor:
        time.sleep(2)
        send_prompt(child1, "What is 5+5?")
        responded = wait_for_text_in_monitor(monitor, "10", timeout=30)
        assert responded, "Slot 3 container should respond"

    # Detach from slot 3 (keeps container running)
    child1.sendcontrol('b')
    time.sleep(0.5)
    child1.send('d')
    time.sleep(2)

    try:
        child1.expect(EOF, timeout=10)
        child1.close()
    except Exception:
        child1.close(force=True)

    time.sleep(2)

    # Launch container on slot 7 with tmux
    child2 = spawn_coi(
        coi_binary,
        ["shell", "--persistent", "--slot=7"],
        cwd=workspace_dir,
        env=env
    )

    wait_for_container_ready(child2, timeout=60)
    wait_for_prompt(child2, timeout=90)

    # Interact to verify it works
    with with_live_screen(child2) as monitor:
        time.sleep(2)
        send_prompt(child2, "What is 7+7?")
        responded = wait_for_text_in_monitor(monitor, "14", timeout=30)
        assert responded, "Slot 7 container should respond"

    # Detach from slot 7
    child2.sendcontrol('b')
    time.sleep(0.5)
    child2.send('d')
    time.sleep(2)

    try:
        child2.expect(EOF, timeout=10)
        child2.close()
    except Exception:
        child2.close(force=True)

    time.sleep(2)

    # Verify both containers are running
    container3 = calculate_container_name(workspace_dir, 3)
    container7 = calculate_container_name(workspace_dir, 7)
    containers = get_container_list()
    assert container3 in containers, f"Container {container3} (slot 3) should be running"
    assert container7 in containers, f"Container {container7} (slot 7) should be running"

    # Now attach to slot 3 specifically (not slot 7)
    child_attach3 = spawn_coi(
        coi_binary,
        ["attach", "--slot=3"],
        cwd=workspace_dir,
        env=env
    )

    wait_for_prompt(child_attach3, timeout=30)

    # Interact to verify we're connected to slot 3
    with with_live_screen(child_attach3) as monitor:
        time.sleep(2)
        send_prompt(child_attach3, "What is 9+9?")
        responded = wait_for_text_in_monitor(monitor, "18", timeout=30)
        assert responded, "Should be able to interact with slot 3 after attach"

        # Exit from slot 3
        clean_exit3 = exit_claude(child_attach3)

    # Now attach to slot 7 specifically
    child_attach7 = spawn_coi(
        coi_binary,
        ["attach", "--slot=7"],
        cwd=workspace_dir,
        env=env
    )

    wait_for_prompt(child_attach7, timeout=30)

    # Interact to verify we're connected to slot 7
    with with_live_screen(child_attach7) as monitor:
        time.sleep(2)
        send_prompt(child_attach7, "What is 11+11?")
        responded = wait_for_text_in_monitor(monitor, "22", timeout=30)
        assert responded, "Should be able to interact with slot 7 after attach"

        # Exit from slot 7
        clean_exit7 = exit_claude(child_attach7)
        wait_for_container_deletion()

    # Verify both exits were clean
    assert_clean_exit(clean_exit3, child_attach3)
    assert_clean_exit(clean_exit7, child_attach7)
