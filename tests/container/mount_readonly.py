"""
Test for coi container mount --readonly - mounts as read-only.

Tests that:
1. Launch a container
2. Mount a directory with --readonly flag
3. Verify writing fails
"""

import os
import subprocess
import tempfile
import time

from support.helpers import (
    calculate_container_name,
)


def test_mount_readonly(coi_binary, cleanup_containers, workspace_dir):
    """
    Test mount with --readonly flag prevents writes.

    Flow:
    1. Create a temp directory with a file
    2. Launch a container
    3. Mount with --readonly flag
    4. Verify reading works but writing fails
    5. Cleanup
    """
    container_name = calculate_container_name(workspace_dir, 1)

    # === Phase 1: Create temp directory with test file ===

    with tempfile.TemporaryDirectory() as tmpdir:
        test_file = os.path.join(tmpdir, "readonly-test.txt")
        with open(test_file, "w") as f:
            f.write("readonly-test-content")

        # === Phase 2: Launch container ===

        result = subprocess.run(
            [coi_binary, "container", "launch", "coi", container_name],
            capture_output=True,
            text=True,
            timeout=120,
        )

        assert result.returncode == 0, \
            f"Container launch should succeed. stderr: {result.stderr}"

        time.sleep(3)

        # === Phase 3: Mount with --readonly ===

        mount_name = "readonly-mount"
        result = subprocess.run(
            [coi_binary, "container", "mount", container_name, tmpdir, "/mnt/readonly", mount_name, "--readonly"],
            capture_output=True,
            text=True,
            timeout=60,
        )

        assert result.returncode == 0, \
            f"Mount with --readonly should succeed. stderr: {result.stderr}"

        time.sleep(2)

        # === Phase 4: Verify read works ===

        result = subprocess.run(
            [coi_binary, "container", "exec", container_name, "--", "cat", "/mnt/readonly/readonly-test.txt"],
            capture_output=True,
            text=True,
            timeout=30,
        )

        assert result.returncode == 0, \
            f"Reading from readonly mount should succeed. stderr: {result.stderr}"

        assert "readonly-test-content" in result.stdout, \
            f"Readonly mount file should contain expected content. Got:\n{result.stdout}"

        # === Phase 5: Verify write fails ===

        result = subprocess.run(
            [coi_binary, "container", "exec", container_name, "--", "touch", "/mnt/readonly/newfile.txt"],
            capture_output=True,
            text=True,
            timeout=30,
        )

        assert result.returncode != 0, \
            "Writing to readonly mount should fail"

        combined_output = result.stdout + result.stderr
        has_error = (
            "read-only" in combined_output.lower() or
            "read only" in combined_output.lower() or
            "permission denied" in combined_output.lower() or
            "cannot" in combined_output.lower()
        )

        assert has_error, \
            f"Should indicate readonly or permission error. Got:\n{combined_output}"

        # === Phase 6: Cleanup ===

        subprocess.run(
            [coi_binary, "container", "delete", container_name, "--force"],
            capture_output=True,
            timeout=30,
        )
