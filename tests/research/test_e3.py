"""E3 deterministic concurrency verification."""

import json
from pathlib import Path

import pytest

from research.e3 import run_e3, run_suite, write_results


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


def test_e3_saved_results_are_reproducible(tmp_path):
    first = run_suite()
    second = run_suite()

    assert first == second
    assert len(first["rows"]) == 4
    assert first["metrics"]["unauthorized_commit_count"] == 1
    assert first["metrics"]["legitimate_successes"] == 2
    assert first["metrics"]["legitimate_attempts"] == 2

    saved = json.loads(
        Path("research/results/e3.json").read_text(encoding="utf-8")
    )
    assert saved == first

    output = tmp_path / "e3.json"
    write_results(first, output)
    assert json.loads(output.read_text(encoding="utf-8")) == saved
