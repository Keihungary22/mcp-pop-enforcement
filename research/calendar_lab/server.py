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

    mcp.run(
        transport="streamable-http",
        host=os.getenv("RESEARCH_MCP_HOST", "127.0.0.1"),
        port=int(os.getenv("RESEARCH_MCP_PORT", "9000")),
        streamable_http_path="/mcp",
        json_response=True,
        stateless_http=True,
    )
