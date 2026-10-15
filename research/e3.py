"""Deterministic check-then-commit TOCTOU experiments."""

import json
import threading
from pathlib import Path

from research.calendar_lab.lab import CalendarLab, PausePoint
from research.runner import ResearchRunner


def run_e3(mode: str, *, swap: bool) -> dict:
    if mode not in ("alias", "immutable"):
        raise ValueError(f"Unsupported execution mode: {mode}")

    lab = CalendarLab()
    runner = ResearchRunner(lab)
    pause = PausePoint()
    invocation_id = f"e3-{mode}-{'swap' if swap else 'control'}"
    outcome = {}
    errors = []

    def worker():
        try:
            outcome["result"] = runner.run(
                invocation_id=invocation_id,
                baseline="B2-strong",
                mode=mode,
                alias="work",
                authorized_resource_id="calendar_A",
                after_mapping_check=pause,
            )
        except Exception as exc:
            errors.append(exc)

    thread = threading.Thread(target=worker, daemon=True)
    thread.start()

    try:
        if not pause.reached.wait(timeout=5):
            raise RuntimeError("Resource-check pause was not reached")
        if swap:
            lab.set_alias("work", "calendar_B")
    finally:
        pause.resume.set()
        thread.join(timeout=5)

    if thread.is_alive():
        raise RuntimeError("Experiment did not terminate")
    if errors:
        raise RuntimeError("Experiment failed") from errors[0]

    decision = runner.authorization_records[0]
    result = outcome["result"]
    commits = lab.commits()
    events = lab.events()
    logs = lab.executions()

    if len(commits) != 1 or len(events) != 1 or len(logs) != 1:
        raise AssertionError("Expected one committed event and execution log")

    target = commits[0]["actual_commit_target"]

    if target != events[0]["resource_id"]:
        raise AssertionError("Commit/event disagreement")
    if target != logs[0]["actual_commit_target"]:
        raise AssertionError("Commit/execution log disagreement")
    if target != result["actual_commit_target"]:
        raise AssertionError("Commit/response disagreement")

    return {
        "experiment": "E3a" if mode == "alias" else "E3b",
        "scenario": "swap" if swap else "control",
        "baseline": "B2-strong",
        "execution_mode": "X1" if mode == "alias" else "X2",
        "authorization_decision": decision["decision"],
        "authorized_resource_id": decision["authorized_resource_id"],
        "alias_at_authorization": decision["alias_at_authorization"],
        "alias_at_execution": logs[0]["alias_at_execution"],
        "actual_commit_target": target,
        "execution_outcome": result["execution_outcome"],
        "unauthorized_commit_count": int(target != decision["authorized_resource_id"]),
        "database_commits": commits,
        "database_events": events,
    }


def run_suite() -> dict:
    rows = [run_e3(mode, swap=swap) for mode in ("alias", "immutable") for swap in (False, True)]
    controls = [row for row in rows if row["scenario"] == "control"]

    return {
        "experiment": "E3",
        "model": "isolated_in_memory_sqlite",
        "rows": rows,
        "metrics": {
            "unauthorized_commit_count": sum(row["unauthorized_commit_count"] for row in rows),
            "legitimate_successes": sum(
                row["authorization_decision"] == "PERMIT"
                and row["execution_outcome"] == "COMMITTED"
                and row["actual_commit_target"] == "calendar_A"
                for row in controls
            ),
            "legitimate_attempts": len(controls),
        },
    }


def write_results(data: dict, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps(data, indent=2) + "\n",
        encoding="utf-8",
    )


if __name__ == "__main__":
    write_results(run_suite(), Path("research/results/e3.json"))
