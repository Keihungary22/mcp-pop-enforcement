"""Mock MCP application logic. / Mock MCPのアプリケーションロジック。"""

_MEETINGS: dict[str, dict[str, str]] = {}


def reset_meetings() -> None:
    """Reset test state. / テスト用状態を初期化する。"""
    _MEETINGS.clear()


def create_meeting(meeting_id: str, title: str) -> dict[str, object]:
    """Create a meeting. / Meetingを作成する。"""
    if meeting_id in _MEETINGS:
        return {
            "ok": False,
            "message": f"Meeting '{meeting_id}' already exists",
        }

    meeting = {
        "meeting_id": meeting_id,
        "title": title,
    }
    _MEETINGS[meeting_id] = meeting

    return {
        "ok": True,
        "message": "Meeting created",
        "meeting": meeting,
    }


def delete_meeting(meeting_id: str) -> dict[str, object]:
    """Delete a meeting. / Meetingを削除する。"""
    meeting = _MEETINGS.pop(meeting_id, None)

    if meeting is None:
        return {
            "ok": False,
            "message": f"Meeting '{meeting_id}' not found",
        }

    return {
        "ok": True,
        "message": "Meeting deleted",
        "meeting": meeting,
    }
