"""Research-only authorization/execution comparison runner."""

from dataclasses import dataclass, field
from typing import Literal

from research.authorization import (
    OPERATION,
    Delegation,
    authorize,
    default_chain,
)
from research.calendar_lab.lab import CalendarLab, Hook

Baseline = Literal["B1", "B2-weak", "B2-strong", "S"]
ExecutionMode = Literal["alias", "immutable"]


@dataclass
class ResearchRunner:
    lab: CalendarLab
    authorization_records: list[dict] = field(default_factory=list)

    def run(
        self,
        *,
        invocation_id: str,
        baseline: Baseline,
        mode: ExecutionMode,
        alias: str = "work",
        authorized_resource_id: str = "calendar_A",
        title: str = "Research event",
        operation: str = OPERATION,
        chain: tuple[Delegation, Delegation] | None = None,
        after_authorization: Hook | None = None,
        after_mapping_check: Hook | None = None,
    ) -> dict:
        if baseline == "S" and mode != "immutable":
            raise ValueError("S requires immutable-ID execution (X2)")

        # Capture mapping before authorization and before any test hook.
        snapshot = self.lab.resolve(alias)

        if chain is None:
            reference = (
                alias
                if baseline == "B2-weak"
                else authorized_resource_id
                if baseline in ("B2-strong", "S")
                else "*"
            )
            chain = default_chain(reference)

        decision = authorize(
            lab=self.lab,
            baseline=baseline,
            operation=operation,
            alias=alias,
            authorized_resource_id=authorized_resource_id,
            chain=chain,
        )

        self.authorization_records.append(
            {
                "invocation_id": invocation_id,
                "baseline": baseline,
                "decision": "PERMIT" if decision.permitted else "DENY",
                "reason": decision.reason,
                "authorized_resource_id": authorized_resource_id,
                "alias_at_authorization": snapshot,
                "operation": operation,
            }
        )

        if not decision.permitted:
            return {
                "invocation_id": invocation_id,
                "authorization_decision": "DENY",
                "execution_outcome": "NOT_EXECUTED",
                "actual_commit_target": None,
            }

        result = self.lab.execute(
            invocation_id=invocation_id,
            alias=alias,
            authorized_resource_id=authorized_resource_id,
            alias_at_authorization=snapshot,
            title=title,
            mode=mode,
            after_authorization=after_authorization,
            after_mapping_check=after_mapping_check,
        )
        return {"authorization_decision": "PERMIT", **result}
