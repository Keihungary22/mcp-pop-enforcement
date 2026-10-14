"""Tests for the isolated deterministic calendar environment."""

import sqlite3
import threading

import pytest

from research.calendar_lab.lab import CalendarLab, PausePoint


def run_scheduled(mode: str, pause_name: str) -> tuple[CalendarLab, dict]:
    lab = CalendarLab()
    pause = PausePoint()
    result = {}
    errors = []

    def invoke() -> None:
        try:
            result.update(
                lab.execute(
                    invocation_id="inv-1",
                    alias="work",
                    authorized_resource_id="calendar_A",
                    title="Research event",
                    mode=mode,
                    **{pause_name: pause},
                )
            )
        except Exception as exc:
            errors.append(exc)

    worker = threading.Thread(target=invoke)
    worker.start()

    try:
        assert pause.reached.wait(5), "Expected synchronization point was not reached"
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


@pytest.mark.parametrize("pause", ["after_authorization", "after_mapping_check"])
@pytest.mark.parametrize(
    ("mode", "expected"),
    [("alias", "calendar_B"), ("immutable", "calendar_A")],
)
def test_deterministic_commit_target(mode: str, expected: str, pause: str) -> None:
    lab, result = run_scheduled(mode, pause)
    assert result["execution_outcome"] == "COMMITTED"
    assert result["actual_commit_target"] == expected
    assert lab.commits()[0]["actual_commit_target"] == expected
    assert lab.commits()[0]["alias_at_execution"] == "calendar_B"
    assert lab.events()[0]["resource_id"] == expected


def test_repeatability() -> None:
    for _ in range(5):
        lab, result = run_scheduled("alias", "after_mapping_check")
        assert result["actual_commit_target"] == "calendar_B"
        assert lab.commits()[0]["alias_at_authorization"] == "calendar_A"


def test_reset() -> None:
    lab = CalendarLab()
    lab.set_alias("work", "calendar_B")
    lab.execute(
        invocation_id="first",
        alias="work",
        authorized_resource_id="calendar_A",
        title="Test",
        mode="alias",
    )
    lab.reset()
    assert lab.resolve("work") == "calendar_A"
    assert lab.commits() == []
    assert lab.events() == []


def test_duplicate_invocation_is_atomic() -> None:
    lab = CalendarLab()
    kwargs = dict(
        invocation_id="duplicate",
        alias="work",
        authorized_resource_id="calendar_A",
        title="Test",
        mode="immutable",
    )
    lab.execute(**kwargs)
    with pytest.raises(sqlite3.IntegrityError):
        lab.execute(**kwargs)
    assert len(lab.events()) == 1
    assert len(lab.commits()) == 1


def test_unknown_resource_is_rejected() -> None:
    lab = CalendarLab()
    with pytest.raises(sqlite3.IntegrityError):
        lab.set_alias("work", "missing")
