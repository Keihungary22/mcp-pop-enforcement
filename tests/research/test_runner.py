"""Authorization and execution comparison tests."""

import pytest

from research.authorization import default_chain
from research.calendar_lab.lab import CalendarLab
from research.runner import ResearchRunner


@pytest.mark.parametrize(
    ("baseline", "mode"),
    [
        ("B1", "alias"),
        ("B2-weak", "alias"),
        ("B2-weak", "immutable"),
        ("B2-strong", "alias"),
        ("B2-strong", "immutable"),
        ("S", "immutable"),
    ],
)
def test_permit_and_commit_are_separate(baseline, mode):
    runner = ResearchRunner(CalendarLab())
    result = runner.run(invocation_id="run-1", baseline=baseline, mode=mode)
    assert result["authorization_decision"] == "PERMIT"
    assert result["execution_outcome"] == "COMMITTED"
    assert result["actual_commit_target"] == "calendar_A"
    assert runner.authorization_records[0]["decision"] == "PERMIT"
    assert len(runner.lab.commits()) == 1


def test_strong_policy_denies_mismatch_without_commit():
    lab = CalendarLab()
    lab.set_alias("work", "calendar_B")
    runner = ResearchRunner(lab)

    result = runner.run(
        invocation_id="deny-1",
        baseline="B2-strong",
        mode="alias",
    )
    assert result["authorization_decision"] == "DENY"
    assert result["execution_outcome"] == "NOT_EXECUTED"
    assert lab.commits() == []
    assert lab.events() == []


def test_weak_alias_constraint_does_not_enforce_immutable_target():
    lab = CalendarLab()
    lab.set_alias("work", "calendar_B")
    runner = ResearchRunner(lab)

    result = runner.run(
        invocation_id="weak-1",
        baseline="B2-weak",
        mode="alias",
    )
    assert result["authorization_decision"] == "PERMIT"
    assert result["actual_commit_target"] == "calendar_B"


def test_strong_authorization_does_not_force_x2():
    runner = ResearchRunner(CalendarLab())

    def swap_alias():
        runner.lab.set_alias("work", "calendar_B")

    result = runner.run(
        invocation_id="strong-x1",
        baseline="B2-strong",
        mode="alias",
        after_authorization=swap_alias,
    )
    assert result["authorization_decision"] == "PERMIT"
    assert result["actual_commit_target"] == "calendar_B"
    assert runner.authorization_records[0]["alias_at_authorization"] == "calendar_A"


def test_s_remains_on_authorized_target():
    runner = ResearchRunner(CalendarLab())

    def swap_alias():
        runner.lab.set_alias("work", "calendar_B")

    result = runner.run(
        invocation_id="strong-x2",
        baseline="S",
        mode="immutable",
        after_authorization=swap_alias,
    )
    assert result["actual_commit_target"] == "calendar_A"


def test_s_rejects_x1_configuration():
    runner = ResearchRunner(CalendarLab())
    with pytest.raises(ValueError, match="S requires"):
        runner.run(invocation_id="invalid", baseline="S", mode="alias")


def test_b1_allows_unrestricted_delegation():
    runner = ResearchRunner(CalendarLab())
    result = runner.run(
        invocation_id="b1-valid",
        baseline="B1",
        mode="alias",
        chain=default_chain("*"),
    )
    assert result["authorization_decision"] == "PERMIT"


def test_resource_restricted_delegation_is_not_treated_as_unrestricted():
    runner = ResearchRunner(CalendarLab())
    result = runner.run(
        invocation_id="b1-restricted",
        baseline="B1",
        mode="alias",
        chain=default_chain("calendar_B"),
    )
    assert result["authorization_decision"] == "DENY"
    assert runner.lab.commits() == []


def test_strong_policy_rejects_incompatible_delegation():
    runner = ResearchRunner(CalendarLab())
    result = runner.run(
        invocation_id="strong-restricted",
        baseline="B2-strong",
        mode="alias",
        chain=default_chain("calendar_B"),
    )
    assert result["authorization_decision"] == "DENY"
    assert runner.lab.commits() == []


def test_unsupported_operation_is_denied():
    runner = ResearchRunner(CalendarLab())
    result = runner.run(
        invocation_id="wrong-op",
        baseline="B2-strong",
        mode="immutable",
        operation="delete_calendar",
    )
    assert result["authorization_decision"] == "DENY"
    assert runner.lab.commits() == []
