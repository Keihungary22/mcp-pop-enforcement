"""Issue #29 deterministic E1/E2 experiment verification."""

import pytest

from research.e1_e2 import CASES, run_case, run_suite


def test_e1_alias_swap_records_actual_commit():
    row = run_case("E1", "B2-weak", "alias", swap=True)

    assert row["authorization_decision"] == "PERMIT"
    assert row["alias_at_authorization"] == "calendar_A"
    assert row["alias_at_execution"] == "calendar_B"
    assert row["unauthorized_commit_count"] == int(
        row["actual_commit_target"] not in (None, "calendar_A")
    )


@pytest.mark.parametrize(("case", "baseline", "mode"), CASES)
@pytest.mark.parametrize("swap", [True, False])
def test_e2_authorization_execution_matrix(case, baseline, mode, swap):
    row = run_case(case, baseline, mode, swap=swap)

    assert row["authorization_decision"] == "PERMIT"
    assert row["authorized_resource_id"] == "calendar_A"
    assert row["alias_at_authorization"] == "calendar_A"
    assert row["alias_at_execution"] == ("calendar_B" if swap else "calendar_A")
    assert row["execution_outcome"] == "COMMITTED"
    assert len(row["database_commits"]) == 1
    assert len(row["database_events"]) == 1

    if not swap:
        assert row["actual_commit_target"] == "calendar_A"
        assert row["legitimate_success"] is True
        assert row["unauthorized_commit_count"] == 0


def test_suite_is_deterministic():
    first = run_suite()
    second = run_suite()

    assert first == second
    assert len(first["experiments"]) == 9
    assert first["metrics"]["legitimate_attempts"] == 4
    assert first["metrics"]["e1_unauthorized_commit_count"] == 1
    assert first["metrics"]["e2_unauthorized_commit_count"] == 2
    assert first["metrics"]["legitimate_successes"] == 4
    assert first["metrics"]["legitimate_operation_success_rate"] == 1.0
