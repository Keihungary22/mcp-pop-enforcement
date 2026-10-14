"""Issue #27: isolated calendar experiment tests."""

import asyncio
import json
import threading

import pytest
from mcp import Client

from research.calendar_lab.lab import CalendarLab, PausePoint
from research.calendar_lab.server import lab as mcp_lab
from research.calendar_lab.server import mcp


def make_args(lab: CalendarLab, invocation_id: str = "inv-1") -> dict:
    return {
        "invocation_id": invocation_id,
        "alias": "work",
        "authorized_resource_id": "calendar_A",
        "alias_at_authorization": lab.resolve("work"),
        "title": "Research event",
    }


def run_scheduled(mode: str, pause_name: str) -> tuple[CalendarLab, dict]:
    lab = CalendarLab()

    # Snapshot is recorded before any experimental alias mutation.
    args = make_args(lab)
    pause = PausePoint()
    result = {}
    errors = []

    def invoke() -> None:
        try:
            result.update(lab.execute(**args, mode=mode, **{pause_name: pause}))
        except Exception as exc:
            errors.append(exc)

    worker = threading.Thread(target=invoke)
    worker.start()
    try:
        assert pause.reached.wait(5), "Synchronization point not reached"
        lab.set_alias("work", "calendar_B")
    finally:
        pause.resume.set()
        worker.join(5)

    assert not worker.is_alive()
    assert not errors, repr(errors)
    return lab, result


def test_fixtures_and_reassignment() -> None:
    lab = CalendarLab()
    assert lab.resolve("work") == "calendar_A"
    lab.create_calendar("calendar_C")
    lab.set_alias("work", "calendar_C")
    assert lab.resolve("work") == "calendar_C"
    with pytest.raises(ValueError):
        lab.set_alias("work", "missing")


@pytest.mark.parametrize("pause", ["after_authorization", "after_mapping_check"])
@pytest.mark.parametrize(
    ("mode", "expected"),
    [("alias", "calendar_B"), ("immutable", "calendar_A")],
)
def test_deterministic_commit_target(mode: str, expected: str, pause: str) -> None:
    lab, result = run_scheduled(mode, pause)
    assert result["execution_outcome"] == "COMMITTED"
    assert result["actual_commit_target"] == expected
    assert lab.events()[0]["resource_id"] == expected
    assert lab.commits()[0]["actual_commit_target"] == expected

    log = lab.executions()[0]
    assert log["authorized_resource_id"] == "calendar_A"
    assert log["alias_at_authorization"] == "calendar_A"
    assert log["alias_at_execution"] == "calendar_B"
    assert log["execution_mode"] == mode
    assert log["execution_outcome"] == "COMMITTED"
    assert log["actual_commit_target"] == expected
    assert lab.version(expected) == 1


def test_repeatability() -> None:
    for _ in range(5):
        lab, result = run_scheduled("alias", "after_mapping_check")
        assert result["actual_commit_target"] == "calendar_B"
        assert lab.executions()[0]["alias_at_authorization"] == "calendar_A"


def test_reset() -> None:
    lab = CalendarLab()
    lab.set_alias("work", "calendar_B")
    lab.execute(**make_args(lab), mode="alias")
    lab.reset()
    assert lab.resolve("work") == "calendar_A"
    assert lab.commits() == []
    assert lab.executions() == []
    assert lab.events() == []
    assert lab.version("calendar_A") == 0


def test_duplicate_does_not_commit_twice() -> None:
    lab = CalendarLab()
    args = make_args(lab, "duplicate")
    lab.execute(**args, mode="immutable")
    with pytest.raises(ValueError, match="Duplicate invocation"):
        lab.execute(**args, mode="immutable")
    assert len(lab.events()) == 1
    assert len(lab.commits()) == 1
    assert lab.version("calendar_A") == 1
    assert [row["execution_outcome"] for row in lab.executions()] == [
        "COMMITTED",
        "FAILED",
    ]


def test_failure_is_recorded_without_commit() -> None:
    lab = CalendarLab()
    with pytest.raises(ValueError, match="Unknown target"):
        args = make_args(lab)
        args["authorized_resource_id"] = "missing"
        lab.execute(**args, mode="immutable")
    assert lab.commits() == []
    assert lab.events() == []
    assert lab.executions()[0]["execution_outcome"] == "FAILED"
    assert lab.executions()[0]["actual_commit_target"] is None


def test_timeout_is_recorded_without_commit() -> None:
    lab = CalendarLab()

    def timeout() -> None:
        raise TimeoutError("Injected pre-execution timeout")

    with pytest.raises(TimeoutError):
        lab.execute(
            **make_args(lab),
            mode="alias",
            after_authorization=timeout,
        )
    assert lab.commits() == []
    assert lab.executions()[0]["execution_outcome"] == "TIMEOUT"


def test_authorization_snapshot_is_not_reconstructed() -> None:
    lab = CalendarLab()
    snapshot = lab.resolve("work")
    lab.set_alias("work", "calendar_B")
    lab.execute(
        invocation_id="snapshot",
        alias="work",
        authorized_resource_id="calendar_A",
        alias_at_authorization=snapshot,
        title="Snapshot test",
        mode="alias",
    )
    log = lab.executions()[0]
    assert log["alias_at_authorization"] == "calendar_A"
    assert log["alias_at_execution"] == "calendar_B"


def test_calendar_tool_over_mcp_protocol() -> None:
    mcp_lab.reset()

    async def run() -> None:
        async with Client(mcp, raise_exceptions=True) as client:
            result = await client.call_tool(
                "run_authorized_calendar_event",
                {
                    "invocation_id": "mcp-1",
                    "baseline": "B2-strong",
                    "mode": "immutable",
                    "alias": "work",
                    "authorized_resource_id": "calendar_A",
                    "title": "MCP protocol test",
                },
            )

            assert result.is_error is False

            body = json.loads(result.content[0].text)

            assert body["authorization_decision"] == "PERMIT"
            assert body["execution_outcome"] == "COMMITTED"
            assert body["actual_commit_target"] == "calendar_A"
            assert mcp_lab.commits()[0]["actual_commit_target"] == "calendar_A"

    asyncio.run(run())
