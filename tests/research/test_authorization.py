"""Tests of minimal research authorization policies."""

import pytest

from research.authorization import (
    Delegation,
    authorize,
    default_chain,
    execution_mode_for_s,
    verify_delegation,
)
from research.calendar_lab.lab import CalendarLab


def test_fixed_delegation_chain() -> None:
    assert verify_delegation(default_chain(), "create_calendar_event", "calendar_A")


def test_wrong_delegation_is_denied() -> None:
    chain = (
        Delegation("human", "agent_a", "create_calendar_event", "calendar_A"),
        Delegation("attacker", "agent_b", "create_calendar_event", "calendar_A"),
    )
    assert not verify_delegation(chain, "create_calendar_event", "calendar_A")


@pytest.mark.parametrize("baseline", ["B1", "B2-weak", "B2-strong", "S"])
def test_valid_authorization(baseline: str) -> None:
    lab = CalendarLab()
    decision = authorize(
        lab=lab,
        baseline=baseline,
        operation="create_calendar_event",
        alias="work",
        authorized_resource_id="calendar_A",
        chain=default_chain(),
    )
    assert decision.permitted


@pytest.mark.parametrize("baseline", ["B2-strong", "S"])
def test_strong_policy_rejects_mismatched_alias(baseline: str) -> None:
    lab = CalendarLab()
    lab.set_alias("work", "calendar_B")
    decision = authorize(
        lab=lab,
        baseline=baseline,
        operation="create_calendar_event",
        alias="work",
        authorized_resource_id="calendar_A",
        chain=default_chain(),
    )
    assert not decision.permitted
    assert decision.reason == "resource_id_mismatch"


def test_weak_policy_does_not_bind_immutable_target() -> None:
    lab = CalendarLab()
    lab.set_alias("work", "calendar_B")
    decision = authorize(
        lab=lab,
        baseline="B2-weak",
        operation="create_calendar_event",
        alias="work",
        authorized_resource_id="calendar_A",
        chain=default_chain(),
    )
    assert decision.permitted


def test_operation_denial() -> None:
    lab = CalendarLab()
    decision = authorize(
        lab=lab,
        baseline="B2-strong",
        operation="delete_calendar",
        alias="work",
        authorized_resource_id="calendar_A",
        chain=default_chain(),
    )
    assert not decision.permitted


def test_s_uses_immutable_execution() -> None:
    assert execution_mode_for_s() == "immutable"
