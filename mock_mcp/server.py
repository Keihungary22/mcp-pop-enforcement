"""Minimal MCP server for development. / 開発用の最小MCP Server。"""

import os

from mcp.server import MCPServer

from mock_mcp.service import (
    create_meeting as create_meeting_record,
)
from mock_mcp.service import (
    delete_meeting as delete_meeting_record,
)

mcp = MCPServer("PoP Mock MCP")


@mcp.tool()
def create_meeting(meeting_id: str, title: str) -> dict[str, object]:
    """Create a test meeting. / テスト用Meetingを作成する。"""
    return create_meeting_record(meeting_id, title)


@mcp.tool()
def delete_meeting(meeting_id: str) -> dict[str, object]:
    """Delete a test meeting. / テスト用Meetingを削除する。"""
    return delete_meeting_record(meeting_id)


if __name__ == "__main__":
    host = os.getenv("MOCK_MCP_HOST", "127.0.0.1")
    port = int(os.getenv("MOCK_MCP_PORT", "9000"))

    mcp.run(
        transport="streamable-http",
        host=host,
        port=port,
        streamable_http_path="/mcp",
        json_response=True,
        stateless_http=True,
    )
