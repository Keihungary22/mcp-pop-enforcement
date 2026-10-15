"""Research policy selection over the MCP protocol."""

import asyncio
import json

from mcp import Client

from research.calendar_lab.server import lab, mcp


def test_mcp_authorization_and_execution_modes() -> None:
    lab.reset()

    async def run() -> None:
        async with Client(mcp, raise_exceptions=True) as client:
            permitted = await client.call_tool(
                "run_authorized_calendar_event",
                {
                    "invocation_id": "mcp-permit",
                    "baseline": "B2-strong",
                    "mode": "immutable",
                    "alias": "work",
                    "authorized_resource_id": "calendar_A",
                    "title": "Permitted",
                },
            )
            assert permitted.is_error is False
            permitted_body = json.loads(permitted.content[0].text)
            assert permitted_body["authorization_decision"] == "PERMIT"
            assert permitted_body["actual_commit_target"] == "calendar_A"

            lab.set_alias("work", "calendar_B")

            denied = await client.call_tool(
                "run_authorized_calendar_event",
                {
                    "invocation_id": "mcp-deny",
                    "baseline": "B2-strong",
                    "mode": "alias",
                    "alias": "work",
                    "authorized_resource_id": "calendar_A",
                    "title": "Denied",
                },
            )
            assert denied.is_error is False
            denied_body = json.loads(denied.content[0].text)
            assert denied_body["authorization_decision"] == "DENY"
            assert denied_body["execution_outcome"] == "NOT_EXECUTED"
            assert len(lab.commits()) == 1

    asyncio.run(run())
