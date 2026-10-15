"""Verify SQLite write-lock behavior using independent connections."""

import sqlite3

import pytest


def test_begin_immediate_prevents_mapping_update_during_checked_write(tmp_path):
    db_path = tmp_path / "e4_atomicity.db"

    with sqlite3.connect(db_path) as setup:
        setup.executescript(
            """
            CREATE TABLE aliases (
                alias TEXT PRIMARY KEY,
                resource_id TEXT NOT NULL
            );
            CREATE TABLE events (
                resource_id TEXT NOT NULL
            );
            INSERT INTO aliases VALUES ('work', 'calendar_A');
            """
        )

    checker = sqlite3.connect(db_path, timeout=0)
    attacker = sqlite3.connect(db_path, timeout=0)

    try:
        checker.execute("BEGIN IMMEDIATE")

        mapping = checker.execute(
            "SELECT resource_id FROM aliases WHERE alias = ?",
            ("work",),
        ).fetchone()[0]
        assert mapping == "calendar_A"

        # A separate connection cannot update the mapping
        # while the checking transaction owns the write reservation.
        with pytest.raises(sqlite3.OperationalError, match="locked"):
            attacker.execute(
                "UPDATE aliases SET resource_id = ? WHERE alias = ?",
                ("calendar_B", "work"),
            )
        attacker.rollback()

        checker.execute(
            "INSERT INTO events (resource_id) VALUES (?)",
            (mapping,),
        )
        checker.commit()

        # After the checked write commits, the mapping may change.
        attacker.execute(
            "UPDATE aliases SET resource_id = ? WHERE alias = ?",
            ("calendar_B", "work"),
        )
        attacker.commit()

        assert checker.execute(
            "SELECT resource_id FROM events"
        ).fetchall() == [("calendar_A",)]
        assert checker.execute(
            "SELECT resource_id FROM aliases WHERE alias = 'work'"
        ).fetchone()[0] == "calendar_B"
    finally:
        checker.close()
        attacker.close()
