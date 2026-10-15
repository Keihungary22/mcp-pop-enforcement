"""E4 existing-control comparison tests."""

import pytest

from research.e4 import MODES, run_e4, run_suite


@pytest.mark.parametrize("mode", MODES)
@pytest.mark.parametrize("swap", [False, True])
def test_e4_deterministic_schedule(mode, swap):
    row = run_e4(mode, swap=swap)

    assert row["authorization_decision"] == "PERMIT"
    assert row["mapping_at_check"] == "calendar_A"

    if not swap:
        assert row["execution_outcome"] == "COMMITTED"
        assert row["actual_commit_target"] == "calendar_A"
        assert row["legitimate_success"]
        assert row["unchanged_mapping_abort_count"] == 0
    elif mode == "transaction_only":
        assert row["execution_outcome"] == "COMMITTED"
        assert row["actual_commit_target"] == "calendar_B"
        assert row["unauthorized_commit_count"] == 1
    elif mode == "immutable":
        assert row["execution_outcome"] == "COMMITTED"
        assert row["actual_commit_target"] == "calendar_A"
        assert row["unauthorized_commit_count"] == 0
    else:
        assert row["execution_outcome"] == "ABORTED"
        assert row["actual_commit_target"] is None
        assert row["database_commits"] == []
        assert row["database_events"] == []
        assert row["aborted_operation_count"] == 1


def test_e4_suite_metrics_and_repeatability():
    a = run_suite()
    b = run_suite()

    assert a == b
    assert len(a["rows"]) == 8
    assert a["metrics"]["unauthorized_commit_count"] == 1
    assert a["metrics"]["target_mismatch_count"] == 1
    assert a["metrics"]["aborted_operation_count"] == 2
    assert a["metrics"]["unchanged_mapping_abort_count"] == 0
    assert a["metrics"]["legitimate_operation_success_rate"] == 1.0
