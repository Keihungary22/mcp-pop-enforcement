"""Minimal research authorization policies, not production token verification."""

from dataclasses import dataclass
from typing import Literal

from research.calendar_lab.lab import CalendarLab

Baseline = Literal["B1", "B2-weak", "B2-strong", "S"]
ExecutionMode = Literal["alias", "immutable"]
OPERATION = "create_calendar_event"


@dataclass(frozen=True)
class Delegation:
    issuer: str
    delegatee: str
    operation: str
    resource_ref: str


@dataclass(frozen=True)
class Decision:
    permitted: bool
    reason: str
    authorized_resource_id: str | None


def default_chain(resource_ref: str = "*") -> tuple[Delegation, Delegation]:
    """Unsigned fixture: human -> agent_a -> agent_b."""
    return (
        Delegation("human", "agent_a", OPERATION, resource_ref),
        Delegation("agent_a", "agent_b", OPERATION, resource_ref),
    )


def verify_delegation(
    chain: tuple[Delegation, Delegation],
    operation: str,
    resource_ref: str,
) -> bool:
    parent, child = chain
    return (
        parent.issuer == "human"
        and parent.delegatee == "agent_a"
        and child.issuer == "agent_a"
        and child.delegatee == "agent_b"
        and parent.operation == operation
        and child.operation == operation
        and parent.resource_ref in ("*", resource_ref)
        and child.resource_ref in ("*", resource_ref)
    )


def authorize(
    *,
    lab: CalendarLab,
    baseline: Baseline,
    operation: str,
    alias: str,
    authorized_resource_id: str,
    chain: tuple[Delegation, Delegation],
) -> Decision:
    """Evaluate independently of the subsequent execution mode."""
    if baseline not in ("B1", "B2-weak", "B2-strong", "S"):
        raise ValueError(f"Unsupported baseline: {baseline}")

    if operation != OPERATION:
        return Decision(False, "operation_not_allowed", None)

    # B1 verifies the delegation structure and operation without a
    # tool-resource constraint. The fixed target remains an evaluation input.
    resource_ref = (
        alias
        if baseline == "B2-weak"
        else authorized_resource_id
        if baseline in ("B2-strong", "S")
        else "*"
    )
    if not verify_delegation(chain, operation, resource_ref):
        return Decision(False, "delegation_invalid", None)

    # The strong policy checks the concrete ID at authorization time.
    if baseline in ("B2-strong", "S"):
        if lab.resolve(alias) != authorized_resource_id:
            return Decision(False, "resource_id_mismatch", None)

    return Decision(True, "permitted", authorized_resource_id)


def execution_mode_for_s() -> ExecutionMode:
    return "immutable"
