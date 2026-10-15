"""Deterministic E1/E2 calendar alias-swap experiments (Issue #29)."""

import json
import threading
from pathlib import Path

from research.calendar_lab.lab import CalendarLab, PausePoint
from research.runner import ResearchRunner

CASES = (
    ("A", "B2-weak", "alias"),
    ("B", "B2-strong", "alias"),
    ("C", "B2-weak", "immutable"),
    ("D", "B2-strong", "immutable"),
)


def run_case(case, baseline, mode, *, swap):
    lab = CalendarLab()
    runner = ResearchRunner(lab)
    pause = PausePoint()
    invocation_id = f"{case}-{'swap' if swap else 'control'}"
    result_box = {}
    errors = []

    def worker():
        try:
            result_box["result"] = runner.run(
                invocation_id=invocation_id,
                baseline=baseline,
                mode=mode,
                alias="work",
                authorized_resource_id="calendar_A",
                title="E1/E2 experiment",
                after_authorization=pause,
            )
        except Exception as exc:
            errors.append(exc)

    thread = threading.Thread(target=worker, daemon=True)
    thread.start()

    try:
        if not pause.reached.wait(timeout=5):
            raise RuntimeError(f"Authorization pause not reached: {invocation_id}")

        if swap:
            lab.set_alias("work", "calendar_B")
    finally:
        pause.resume.set()
        thread.join(timeout=5)

    if thread.is_alive():
        raise RuntimeError(f"Experiment thread did not finish: {invocation_id}")
    if errors:
        raise RuntimeError(f"Experiment failed: {invocation_id}") from errors[0]

    response = result_box["result"]
    authorization = runner.authorization_records[0]
    commits = [row for row in lab.commits() if row["invocation_id"] == invocation_id]
    events = [row for row in lab.events() if row["invocation_id"] == invocation_id]
    execution_logs = [row for row in lab.executions() if row["invocation_id"] == invocation_id]

    if len(execution_logs) != 1:
        raise AssertionError("Expected exactly one execution log")

    actual_target = commits[0]["actual_commit_target"] if commits else None
    authorized_target = authorization["authorized_resource_id"]

    # The event table is independent evidence of the committed target.
    if len(events) != len(commits):
        raise AssertionError("Event/commit count mismatch")
    if commits and events[0]["resource_id"] != actual_target:
        raise AssertionError("Event target differs from commit log")
    if response["actual_commit_target"] != actual_target:
        raise AssertionError("Response differs from committed database state")
    if execution_logs[0]["actual_commit_target"] != actual_target:
        raise AssertionError("Execution log differs from commit log")

    unauthorized_count = sum(row["actual_commit_target"] != authorized_target for row in commits)

    return {
        "experiment": "E1" if case == "E1" else "E2",
        "case": case,
        "scenario": "alias_swap" if swap else "legitimate_control",
        "baseline": baseline,
        "execution_mode": "X1" if mode == "alias" else "X2",
        "invocation_id": invocation_id,
        "authorization_decision": authorization["decision"],
        "authorized_resource_id": authorized_target,
        "alias_at_authorization": authorization["alias_at_authorization"],
        "alias_at_execution": execution_logs[0]["alias_at_execution"],
        "execution_outcome": response["execution_outcome"],
        "actual_commit_target": actual_target,
        "unauthorized_commit_count": unauthorized_count,
        "legitimate_success": (
            not swap
            and authorization["decision"] == "PERMIT"
            and response["execution_outcome"] == "COMMITTED"
            and actual_target == authorized_target
        ),
        "database_commits": commits,
        "database_events": events,
        "execution_logs": execution_logs,
    }


def run_suite():
    results = [run_case("E1", "B2-weak", "alias", swap=True)]

    for case, baseline, mode in CASES:
        results.append(run_case(case, baseline, mode, swap=True))
        results.append(run_case(case, baseline, mode, swap=False))

    controls = [row for row in results if row["scenario"] == "legitimate_control"]
    successful = sum(row["legitimate_success"] for row in controls)

    return {
        "issue": 29,
        "policy": "fixed_authorized_target",
        "model": "isolated_in_memory_sqlite",
        "experiments": results,
        "metrics": {
            "e1_unauthorized_commit_count": sum(
                r["unauthorized_commit_count"] for r in results if r["experiment"] == "E1"
            ),
            "e2_unauthorized_commit_count": sum(
                r["unauthorized_commit_count"] for r in results if r["experiment"] == "E2"
            ),
            "legitimate_operation_success_rate": (successful / len(controls)),
            "legitimate_successes": successful,
            "legitimate_attempts": len(controls),
        },
    }


def write_results(data):
    # Validate conclusions before creating or overwriting result files.
    rows = data["experiments"]
    metrics = data["metrics"]

    expected = {
        (baseline, "X1" if mode == "alias" else "X2"): (
            "calendar_B" if mode == "alias" else "calendar_A",
            int(mode == "alias"),
        )
        for _, baseline, mode in CASES
    }

    swaps = [
        row for row in rows
        if row["experiment"] == "E2" and row["scenario"] == "alias_swap"
    ]

    seen = set()
    for row in swaps:
        key = (row["baseline"], row["execution_mode"])
        if key not in expected or key in seen:
            raise ValueError(f"Unexpected or duplicate E2 swap case: {key}")
        seen.add(key)

        target, unauthorized = expected[key]
        if (
            row["authorization_decision"] != "PERMIT"
            or row["execution_outcome"] != "COMMITTED"
            or row["actual_commit_target"] != target
            or row["unauthorized_commit_count"] != unauthorized
            or len(row["database_commits"]) != 1
            or len(row["database_events"]) != 1
            or row["database_commits"][0]["actual_commit_target"] != target
            or row["database_events"][0]["resource_id"] != target
        ):
            raise ValueError(f"E2 swap observation contradicts findings: {key}")

    if seen != set(expected):
        raise ValueError("Incomplete E2 swap matrix")

    controls = [
        row for row in rows
        if row["experiment"] == "E2"
        and row["scenario"] == "legitimate_control"
    ]

    if (
        len(controls) != 4
        or any(
            not row["legitimate_success"]
            or row["authorization_decision"] != "PERMIT"
            or row["execution_outcome"] != "COMMITTED"
            or row["actual_commit_target"] != "calendar_A"
            for row in controls
        )
        or metrics["legitimate_attempts"] != 4
        or metrics["legitimate_successes"] != 4
        or metrics["legitimate_operation_success_rate"] != 1.0
    ):
        raise ValueError("Legitimate-control observations contradict findings")

    if (
        metrics["e1_unauthorized_commit_count"] != sum(
            row["unauthorized_commit_count"]
            for row in rows if row["experiment"] == "E1"
        )
        or metrics["e2_unauthorized_commit_count"] != sum(
            row["unauthorized_commit_count"]
            for row in swaps
        )
    ):
        raise ValueError("Reported unauthorized-commit metrics do not match rows")

    directory = Path("research/results")
    directory.mkdir(parents=True, exist_ok=True)

    json_path = directory / "e1_e2.json"
    json_path.write_text(
        json.dumps(data, indent=2, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )

    rows = data["experiments"]
    lines = [
        "# Issue #29 — E1/E2 Initial Findings",
        "",
        "## Experimental conditions",
        "",
        "- Fixed authorized target: `calendar_A`.",
        "- Initial mapping: `work -> calendar_A`.",
        "- Swap scenarios: mapping changes to `calendar_B` after authorization.",
        "- Control scenarios: mapping remains unchanged.",
        "- Synchronization: `PausePoint`; no timing-based race assumptions.",
        "- Evidence: SQLite event rows, commit logs and execution logs.",
        "",
        "## Observed results",
        "",
        (
            "| Experiment | Case | Baseline | Mode | Scenario | Decision "
            "| Outcome | Actual target | Unauthorized commits |"
        ),
        "| --- | --- | --- | --- | --- | --- | --- | --- | ---: |",
    ]

    for row in rows:
        lines.append(
            f"| {row['experiment']} | {row['case']} | {row['baseline']} "
            f"| {row['execution_mode']} | {row['scenario']} "
            f"| {row['authorization_decision']} "
            f"| {row['execution_outcome']} "
            f"| {row['actual_commit_target'] or 'None'} "
            f"| {row['unauthorized_commit_count']} |"
        )

    metrics = data["metrics"]
    lines.extend(
        [
            "",
            "## Metrics",
            "",
            f"- E1 Unauthorized Commits: **{metrics['e1_unauthorized_commit_count']}**",
            f"- E2 Unauthorized Commits: **{metrics['e2_unauthorized_commit_count']}**",
            "- E1 and E2-A repeat the same configuration; results are not independent.",
            (
                "- Legitimate Operation Success Rate: "
                f"**{metrics['legitimate_successes']}/"
                f"{metrics['legitimate_attempts']} "
                f"({metrics['legitimate_operation_success_rate']:.0%})**"
            ),
            "",
            "## Feasibility assessment",
            "",
            "These measurements characterize the implemented research model only.",
            "A mismatch in B2-weak + X1 is a negative-control result,",
            "not evidence of research novelty.",
            "The combined B2-strong + X2 configuration represents the",
            "strong existing control for this comparison.",
            "",
            "E3/E4 may be justified by a separately identified question about",
            "execution boundaries or concurrency; this experiment alone",
            "does not establish that such a question is novel.",
            "",
            "## Preliminary research decision: REFINE",
            "",
            "Decision: **Refine (provisional)**.",
            "",
            "Both X1 configurations committed to calendar_B after authorization",
            "for calendar_A. Both X2 configurations preserved calendar_A.",
            "Authorization-time ID validation alone did not preserve the target",
            "when execution independently re-resolved the mutable alias.",
            "",
            "B2-strong + X1 does not invalidate correctly maintained immutable-ID",
            "binding: this configuration discards the verified ID at execution.",
            "These results demonstrate required consistency conditions,",
            "not a novel vulnerability or security mechanism.",
            "",
            "The legitimate controls succeeded in 4/4 tested cases (100%).",
            "This does not establish safety under arbitrary concurrency.",
            "",
            "Before E3/E4, define which additional authorization condition",
            "can change and what atomicity is required to preserve it.",
            "Reassess novelty against existing research.",
            "",
            "## Limitations",
            "",
            "- The delegation chain is an unsigned fixture.",
            "- This suite directly invokes the research runner, not the Go Gateway.",
            "- The calendar database is isolated and in-memory.",
            "- Experiment settings are controlled by the test harness.",
            "- No production-level security claim follows from these results.",
            "- Continue / Refine / Early No-Go requires interpreting the",
            "  observations against existing research and security controls.",
            "",
        ]
    )

    report_path = directory / "e1_e2_findings.md"
    report_path.write_text("\n".join(lines), encoding="utf-8")
    return json_path, report_path


if __name__ == "__main__":
    data = run_suite()
    paths = write_results(data)

    for row in data["experiments"]:
        print(
            f"{row['case']:>2} {row['scenario']:<18} "
            f"{row['baseline']:<10} {row['execution_mode']} "
            f"{row['authorization_decision']:<6} "
            f"target={row['actual_commit_target']} "
            f"unauthorized={row['unauthorized_commit_count']}"
        )

    print("Metrics:", data["metrics"])
    for path in paths:
        print("Created:", path)
