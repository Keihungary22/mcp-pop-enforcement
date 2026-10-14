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
    socket_path = os.getenv("MOCK_MCP_SOCKET")

    if socket_path:
        import socket

        import uvicorn

        app = mcp.streamable_http_app(
            streamable_http_path="/mcp",
            json_response=True,
            stateless_http=True,
            host="mock-mcp",
        )

        if os.path.lexists(socket_path):
            import errno
            import stat

            existing = os.lstat(socket_path)

            if not stat.S_ISSOCK(existing.st_mode):
                raise RuntimeError(f"Socket path is occupied by a non-socket: {socket_path}")

            with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as probe:
                probe.settimeout(2)

                try:
                    probe.connect(socket_path)
                except OSError as exc:
                    if exc.errno != errno.ECONNREFUSED:
                        raise
                else:
                    raise RuntimeError(f"Socket is already in use: {socket_path}")

            os.unlink(socket_path)

        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as sock:
            sock.bind(socket_path)
            os.chown(socket_path, 0, 101)
            os.chmod(socket_path, 0o660)
            sock.listen(128)

            uvicorn.run(app, fd=sock.fileno())
    else:
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
