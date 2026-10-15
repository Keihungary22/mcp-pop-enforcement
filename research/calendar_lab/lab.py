"""Isolated deterministic calendar experiment environment (Issue #27)."""

import json
import sqlite3
import threading
from collections.abc import Callable
from typing import Literal

Mode = Literal["alias", "immutable"]
Hook = Callable[[], None]


class PausePoint:
    """A synchronization barrier controlled explicitly by the test."""

    def __init__(self) -> None:
        self.reached = threading.Event()
        self.resume = threading.Event()

    def __call__(self) -> None:
        self.reached.set()
        if not self.resume.wait(timeout=10):
            raise TimeoutError("Synchronization point was not released")


class CalendarLab:
    def __init__(self) -> None:
        self.lock = threading.RLock()
        self.db = sqlite3.connect(":memory:", check_same_thread=False)
        self.db.row_factory = sqlite3.Row
        self.db.execute("PRAGMA foreign_keys = ON")
        self.reset()

    def reset(self) -> None:
        with self.lock:
            self.db.executescript("""
                DROP TABLE IF EXISTS execution_logs;
                DROP TABLE IF EXISTS commit_logs;
                DROP TABLE IF EXISTS events;
                DROP TABLE IF EXISTS aliases;
                DROP TABLE IF EXISTS calendars;

                CREATE TABLE calendars (
                    resource_id TEXT PRIMARY KEY,
                    version INTEGER NOT NULL DEFAULT 0
                );

                CREATE TABLE aliases (
                    alias TEXT PRIMARY KEY,
                    resource_id TEXT NOT NULL REFERENCES calendars(resource_id)
                );

                CREATE TABLE events (
                    invocation_id TEXT PRIMARY KEY,
                    resource_id TEXT NOT NULL REFERENCES calendars(resource_id),
                    title TEXT NOT NULL
                );

                CREATE TABLE commit_logs (
                    invocation_id TEXT PRIMARY KEY,
                    actual_commit_target TEXT NOT NULL
                        REFERENCES calendars(resource_id),
                    operation TEXT NOT NULL
                );

                CREATE TABLE execution_logs (
                    log_id INTEGER PRIMARY KEY AUTOINCREMENT,
                    invocation_id TEXT NOT NULL,
                    requested_tool TEXT NOT NULL,
                    arguments_json TEXT NOT NULL,
                    authorized_resource_id TEXT NOT NULL,
                    alias_at_authorization TEXT NOT NULL,
                    alias_at_execution TEXT,
                    execution_mode TEXT NOT NULL,
                    execution_outcome TEXT NOT NULL,
                    actual_commit_target TEXT,
                    error_type TEXT
                );
            """)
            self.db.executemany(
                "INSERT INTO calendars(resource_id) VALUES (?)",
                [("calendar_A",), ("calendar_B",)],
            )
            self.db.execute(
                "INSERT INTO aliases(alias, resource_id) VALUES (?, ?)",
                ("work", "calendar_A"),
            )
            self.db.commit()

    def create_calendar(self, resource_id: str) -> None:
        with self.lock, self.db:
            self.db.execute(
                "INSERT INTO calendars(resource_id) VALUES (?)",
                (resource_id,),
            )

    def resolve(self, alias: str) -> str:
        with self.lock:
            row = self.db.execute(
                "SELECT resource_id FROM aliases WHERE alias = ?",
                (alias,),
            ).fetchone()
            if row is None:
                raise ValueError(f"Unknown alias: {alias}")
            return str(row["resource_id"])

    def set_alias(self, alias: str, resource_id: str) -> None:
        with self.lock, self.db:
            if (
                self.db.execute(
                    "SELECT 1 FROM calendars WHERE resource_id = ?",
                    (resource_id,),
                ).fetchone()
                is None
            ):
                raise ValueError(f"Unknown calendar: {resource_id}")

            self.db.execute(
                """INSERT INTO aliases(alias, resource_id) VALUES (?, ?)
                   ON CONFLICT(alias)
                   DO UPDATE SET resource_id = excluded.resource_id""",
                (alias, resource_id),
            )

    def _record_execution(
        self,
        *,
        invocation_id: str,
        alias: str,
        title: str,
        authorized_resource_id: str,
        alias_at_authorization: str,
        alias_at_execution: str | None,
        mode: str,
        outcome: str,
        target: str | None,
        error_type: str | None,
    ) -> None:
        with self.lock, self.db:
            self.db.execute(
                """INSERT INTO execution_logs (
                    invocation_id, requested_tool, arguments_json,
                    authorized_resource_id, alias_at_authorization,
                    alias_at_execution, execution_mode, execution_outcome,
                    actual_commit_target, error_type
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)""",
                (
                    invocation_id,
                    "create_calendar_event",
                    json.dumps({"alias": alias, "title": title}, sort_keys=True),
                    authorized_resource_id,
                    alias_at_authorization,
                    alias_at_execution,
                    mode,
                    outcome,
                    target,
                    error_type,
                ),
            )

    def execute(
        self,
        *,
        invocation_id: str,
        alias: str,
        authorized_resource_id: str,
        alias_at_authorization: str,
        title: str,
        mode: Mode,
        after_authorization: Hook | None = None,
        after_mapping_check: Hook | None = None,
    ) -> dict[str, str]:
        """Execute a caller-supplied authorization snapshot; no policy checks.

        The caller must capture alias_at_authorization at authorization time.
        This method never reconstructs that historical value from live state.
        """
        execution_mapping = None
        try:
            if mode not in ("alias", "immutable"):
                raise ValueError(f"Unknown execution mode: {mode}")

            if after_authorization is not None:
                after_authorization()

            # Explicit non-atomic check for the future E3 TOCTOU experiment.
            self.resolve(alias)

            if after_mapping_check is not None:
                after_mapping_check()

            with self.lock:
                execution_mapping = self.resolve(alias)
                target = execution_mapping if mode == "alias" else authorized_resource_id

                with self.db:
                    # Reject duplicate invocation IDs before writing.
                    if (
                        self.db.execute(
                            "SELECT 1 FROM commit_logs WHERE invocation_id = ?",
                            (invocation_id,),
                        ).fetchone()
                        is not None
                    ):
                        raise ValueError(f"Duplicate invocation ID: {invocation_id}")

                    exists = self.db.execute(
                        "SELECT 1 FROM calendars WHERE resource_id = ?",
                        (target,),
                    ).fetchone()
                    if exists is None:
                        raise ValueError(f"Unknown target resource: {target}")

                    self.db.execute(
                        "INSERT INTO events VALUES (?, ?, ?)",
                        (invocation_id, target, title),
                    )
                    updated = self.db.execute(
                        """UPDATE calendars
                           SET version = version + 1
                           WHERE resource_id = ?""",
                        (target,),
                    )
                    if updated.rowcount != 1:
                        raise RuntimeError("Calendar version update failed")

                    self.db.execute(
                        "INSERT INTO commit_logs VALUES (?, ?, ?)",
                        (invocation_id, target, "create_calendar_event"),
                    )

            # The database transaction has completed successfully here.
            self._record_execution(
                invocation_id=invocation_id,
                alias=alias,
                title=title,
                authorized_resource_id=authorized_resource_id,
                alias_at_authorization=alias_at_authorization,
                alias_at_execution=execution_mapping,
                mode=mode,
                outcome="COMMITTED",
                target=target,
                error_type=None,
            )
            return {
                "invocation_id": invocation_id,
                "execution_outcome": "COMMITTED",
                "actual_commit_target": target,
            }

        except Exception as exc:
            self._record_execution(
                invocation_id=invocation_id,
                alias=alias,
                title=title,
                authorized_resource_id=authorized_resource_id,
                alias_at_authorization=alias_at_authorization,
                alias_at_execution=execution_mapping,
                mode=mode,
                outcome="TIMEOUT" if isinstance(exc, TimeoutError) else "FAILED",
                target=None,
                error_type=type(exc).__name__,
            )
            raise

    def commits(self) -> list[dict]:
        with self.lock:
            return [
                dict(row)
                for row in self.db.execute("SELECT * FROM commit_logs ORDER BY rowid").fetchall()
            ]

    def executions(self) -> list[dict]:
        with self.lock:
            return [
                dict(row)
                for row in self.db.execute(
                    "SELECT * FROM execution_logs ORDER BY log_id"
                ).fetchall()
            ]

    def events(self) -> list[dict]:
        with self.lock:
            return [
                dict(row)
                for row in self.db.execute("SELECT * FROM events ORDER BY rowid").fetchall()
            ]

    def version(self, resource_id: str) -> int:
        with self.lock:
            row = self.db.execute(
                "SELECT version FROM calendars WHERE resource_id = ?",
                (resource_id,),
            ).fetchone()
            if row is None:
                raise ValueError(f"Unknown resource: {resource_id}")
            return int(row["version"])
