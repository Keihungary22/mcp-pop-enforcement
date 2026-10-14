"""Isolated deterministic calendar experiment environment."""

import json
import sqlite3
import threading
from collections.abc import Callable
from typing import Literal

Mode = Literal["alias", "immutable"]
Hook = Callable[[], None]


class PausePoint:
    """A deterministic synchronization point controlled by a test."""

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
                    requested_tool TEXT NOT NULL,
                    arguments_json TEXT NOT NULL,
                    authorized_resource_id TEXT NOT NULL,
                    alias_at_authorization TEXT NOT NULL,
                    alias_at_execution TEXT NOT NULL,
                    execution_mode TEXT NOT NULL,
                    execution_outcome TEXT NOT NULL,
                    actual_commit_target TEXT NOT NULL
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
            self.db.execute(
                """INSERT INTO aliases(alias, resource_id)
                   VALUES (?, ?)
                   ON CONFLICT(alias)
                   DO UPDATE SET resource_id = excluded.resource_id""",
                (alias, resource_id),
            )

    def execute(
        self,
        *,
        invocation_id: str,
        alias: str,
        authorized_resource_id: str,
        title: str,
        mode: Mode,
        after_authorization: Hook | None = None,
        after_mapping_check: Hook | None = None,
    ) -> dict[str, str]:
        """Perform execution only; authorization belongs to Issue #28."""
        if mode not in ("alias", "immutable"):
            raise ValueError(f"Unknown execution mode: {mode}")

        authorization_mapping = self.resolve(alias)

        if after_authorization:
            after_authorization()

        self.resolve(alias)

        if after_mapping_check:
            after_mapping_check()

        with self.lock:
            target = self.resolve(alias) if mode == "alias" else authorized_resource_id
            execution_mapping = self.resolve(alias)

            with self.db:
                self.db.execute(
                    "INSERT INTO events VALUES (?, ?, ?)",
                    (invocation_id, target, title),
                )
                self.db.execute(
                    "UPDATE calendars SET version = version + 1 WHERE resource_id = ?",
                    (target,),
                )
                self.db.execute(
                    "INSERT INTO commit_logs VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
                    (
                        invocation_id,
                        "create_calendar_event",
                        json.dumps({"alias": alias, "title": title}, sort_keys=True),
                        authorized_resource_id,
                        authorization_mapping,
                        execution_mapping,
                        mode,
                        "COMMITTED",
                        target,
                    ),
                )

        return {
            "invocation_id": invocation_id,
            "execution_outcome": "COMMITTED",
            "actual_commit_target": target,
        }

    def commits(self) -> list[dict]:
        with self.lock:
            return [
                dict(row)
                for row in self.db.execute("SELECT * FROM commit_logs ORDER BY rowid").fetchall()
            ]

    def events(self) -> list[dict]:
        with self.lock:
            return [
                dict(row)
                for row in self.db.execute("SELECT * FROM events ORDER BY rowid").fetchall()
            ]
