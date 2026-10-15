"""Research-only MCP server, not registered in the baseline."""

from mcp.server import MCPServer

from research.calendar_lab.lab import CalendarLab
from research.runner import ResearchRunner

lab = CalendarLab()
runner = ResearchRunner(lab)
mcp = MCPServer("Calendar Research Lab")


def create_calendar_event(
    invocation_id: str,
    alias: str,
    authorized_resource_id: str,
    alias_at_authorization: str,
    title: str,
    mode: str,
) -> dict[str, str]:
    """Execute a research invocation without performing authorization."""
    return lab.execute(
        invocation_id=invocation_id,
        alias=alias,
        authorized_resource_id=authorized_resource_id,
        alias_at_authorization=alias_at_authorization,
        title=title,
        mode=mode,
    )


@mcp.tool()
def run_authorized_calendar_event(
    invocation_id: str,
    baseline: str,
    mode: str,
    alias: str = "work",
    authorized_resource_id: str = "calendar_A",
    title: str = "Research event",
) -> dict:
    """Run a research policy fixture after gateway OAuth/DPoP validation.

    Delegation is a fixed unsigned fixture, not authenticated user delegation.
    The supplied target is experimental input, not trusted token authority.
    """
    return runner.run(
        invocation_id=invocation_id,
        baseline=baseline,
        mode=mode,
        alias=alias,
        authorized_resource_id=authorized_resource_id,
        title=title,
    )


if __name__ == "__main__":
    import os

    socket_path = os.getenv("RESEARCH_MCP_SOCKET")

    if socket_path:
        import socket

        import uvicorn

        app = mcp.streamable_http_app(
            streamable_http_path="/mcp",
            json_response=True,
            stateless_http=True,
            host="research-mcp",
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
        host = os.getenv("RESEARCH_MCP_HOST", "127.0.0.1")
        port = int(os.getenv("RESEARCH_MCP_PORT", "9000"))

        mcp.run(
            transport="streamable-http",
            host=host,
            port=port,
            streamable_http_path="/mcp",
            json_response=True,
            stateless_http=True,
        )
