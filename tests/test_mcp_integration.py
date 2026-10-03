import asyncio

from mcp import Client

from mock_mcp.server import mcp
from mock_mcp.service import reset_meetings


def test_create_meeting_through_mcp_protocol() -> None:
    reset_meetings()

    async def run() -> None:
        async with Client(mcp, raise_exceptions=True) as client:
            result = await client.call_tool(
                "create_meeting",
                {
                    "meeting_id": "integration-1",
                    "title": "MCP Integration Test",
                },
            )

            assert result.is_error is False
            assert result.structured_content is not None
            assert result.structured_content["ok"] is True

    asyncio.run(run())
