"""E3 deterministic concurrency verification."""

import pytest

from research.e3 import run_e3


@pytest.mark.parametrize("mode", ["alias", "immutable"])
@pytest.mark.parametrize("swap", [False, True])
def test_e3_schedule(mode, swap):
    row = run_e3(mode, swap=swap)

    assert row["authorization_decision"] == "PERMIT"
    assert row["authorized_resource_id"] == "calendar_A"
    assert row["alias_at_authorization"] == "calendar_A"
    assert row["alias_at_execution"] == ("calendar_B" if swap else "calendar_A")
    assert row["execution_outcome"] == "COMMITTED"

    expected_target = "calendar_B" if swap and mode == "alias" else "calendar_A"

    assert row["actual_commit_target"] == expected_target
    assert row["unauthorized_commit_count"] == int(expected_target != "calendar_A")
