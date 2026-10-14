"""Research-only MCP server, separate from the baseline MCP server."""

from mcp.server import MCPServer

from research.calendar_lab.lab import CalendarLab

lab = CalendarLab()
mcp = MCPServer("Calendar Research Lab")


@mcp.tool()
def create_calendar_event(
    invocation_id: str,
    alias: str,
    authorized_resource_id: str,
    title: str,
    mode: str,
) -> dict[str, str]:
    """Execute an experimental calendar operation."""
    return lab.execute(
        invocation_id=invocation_id,
        alias=alias,
        authorized_resource_id=authorized_resource_id,
        title=title,
        mode=mode,
    )


if __name__ == "__main__":
    mcp.run(transport="stdio")
