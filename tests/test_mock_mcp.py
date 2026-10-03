from mock_mcp.service import create_meeting, delete_meeting, reset_meetings


def setup_function() -> None:
    reset_meetings()


def test_create_meeting() -> None:
    result = create_meeting("meeting-1", "Security Lab")

    assert result["ok"] is True
    assert result["meeting"]["meeting_id"] == "meeting-1"


def test_delete_meeting() -> None:
    create_meeting("meeting-1", "Security Lab")

    result = delete_meeting("meeting-1")

    assert result["ok"] is True


def test_delete_unknown_meeting() -> None:
    result = delete_meeting("missing")

    assert result["ok"] is False
