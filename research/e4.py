"""E4: deterministic comparison of existing execution consistency controls."""

import threading

from research.authorization import authorize, default_chain
from research.calendar_lab.lab import CalendarLab, PausePoint

MODES = ("immutable", "mapping_cas", "checked_transaction", "transaction_only")


def run_e4(mode: str, *, swap: bool) -> dict:
    if mode not in MODES:
        raise ValueError(f"Unsupported mode: {mode}")

    lab = CalendarLab()
    invocation_id = f"e4-{mode}-{'swap' if swap else 'control'}"
    authorized_id = "calendar_A"
    order = []

    decision = authorize(
        lab=lab,
        baseline="B2-strong",
        operation="create_calendar_event",
        alias="work",
        authorized_resource_id=authorized_id,
        chain=default_chain(authorized_id),
    )
    assert decision.permitted
    order.append("authorization")

    # Initial check is deliberately outside the commit transaction.
    checked_mapping = lab.resolve("work")
    assert checked_mapping == authorized_id
    order.append("resource_check")

    pause = PausePoint()
    outcome = {}
    errors = []

    def worker():
        try:
            pause()
            order.append("execution")

            with lab.lock, lab.db:
                lab.db.execute("BEGIN IMMEDIATE")
                current = lab.resolve("work")

                if mode == "immutable":
                    target = authorized_id
                elif mode == "transaction_only":
                    target = current
                elif mode == "checked_transaction":
                    target = authorized_id if current == authorized_id else None
                else:
                    target = authorized_id

                if mode == "mapping_cas":
                    # Conditional SQL write: the mapping must still match.
                    inserted = lab.db.execute(
                        """INSERT INTO events (invocation_id, resource_id, title)
                           SELECT ?, ?, ? FROM aliases
                           WHERE alias = ? AND resource_id = ?""",
                        (
                            invocation_id,
                            authorized_id,
                            "E4 comparison",
                            "work",
                            authorized_id,
                        ),
                    )
                    if inserted.rowcount != 1:
                        target = None
                elif target is not None:
                    lab.db.execute(
                        "INSERT INTO events VALUES (?, ?, ?)",
                        (invocation_id, target, "E4 comparison"),
                    )

                if target is not None:
                    updated = lab.db.execute(
                        "UPDATE calendars SET version = version + 1 WHERE resource_id = ?",
                        (target,),
                    )
                    if updated.rowcount != 1:
                        raise RuntimeError("Calendar update failed")
                    lab.db.execute(
                        "INSERT INTO commit_logs VALUES (?, ?, ?)",
                        (invocation_id, target, "create_calendar_event"),
                    )

            outcome["target"] = target
            outcome["status"] = "COMMITTED" if target else "ABORTED"
            order.append("commit" if target else "abort")
        except Exception as exc:
            errors.append(exc)

    thread = threading.Thread(target=worker, daemon=True)
    thread.start()

    try:
        if not pause.reached.wait(timeout=5):
            raise TimeoutError("Execution synchronization failed")
        if swap:
            lab.set_alias("work", "calendar_B")
            order.append("alias_change")
    finally:
        pause.resume.set()
        thread.join(timeout=5)

    if thread.is_alive():
        raise TimeoutError("Worker did not terminate")
    if errors:
        raise RuntimeError("E4 worker failed") from errors[0]

    commits = lab.commits()
    events = lab.events()
    target = outcome["target"]

    assert len(commits) == len(events) == (1 if target else 0)
    if target:
        assert commits[0]["actual_commit_target"] == target
        assert events[0]["resource_id"] == target

    assert order[:2] == ["authorization", "resource_check"]
    assert order[-1] in ("commit", "abort")
    assert ("alias_change" in order) == swap

    return {
        "mode": mode,
        "scenario": "swap" if swap else "control",
        "schedule": order,
        "authorization_decision": "PERMIT",
        "authorized_resource_id": authorized_id,
        "mapping_at_check": checked_mapping,
        "mapping_at_execution": lab.resolve("work"),
        "execution_outcome": outcome["status"],
        "actual_commit_target": target,
        "unauthorized_commit_count": int(target is not None and target != authorized_id),
        "target_mismatch_count": int(target is not None and target != authorized_id),
        "aborted_operation_count": int(target is None),
        "unchanged_mapping_abort_count": int(not swap and target is None),
        "legitimate_success": not swap and target == authorized_id,
        "database_commits": commits,
        "database_events": events,
    }


def run_suite() -> dict:
    rows = [run_e4(mode, swap=swap) for mode in MODES for swap in (False, True)]
    controls = [row for row in rows if row["scenario"] == "control"]

    return {
        "experiment": "E4",
        "rows": rows,
        "metrics": {
            "unauthorized_commit_count": sum(r["unauthorized_commit_count"] for r in rows),
            "target_mismatch_count": sum(r["target_mismatch_count"] for r in rows),
            "aborted_operation_count": sum(r["aborted_operation_count"] for r in rows),
            "unchanged_mapping_abort_count": sum(r["unchanged_mapping_abort_count"] for r in rows),
            "legitimate_operation_success_rate": (
                sum(r["legitimate_success"] for r in controls) / len(controls)
            ),
        },
    }


if __name__ == "__main__":
    import json
    from pathlib import Path

    data = run_suite()
    path = Path("research/results/e4.json")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps(data, indent=2) + "\n",
        encoding="utf-8",
    )

    for row in data["rows"]:
        print(
            f"{row['mode']:<20} {row['scenario']:<8} "
            f"{row['execution_outcome']:<9} "
            f"target={row['actual_commit_target']}"
        )
    print("Metrics:", data["metrics"])
    print("Created:", path)
