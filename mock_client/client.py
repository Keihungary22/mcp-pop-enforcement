"""Mock client agent for development. / 開発用Mock Client Agent。"""

from __future__ import annotations

import argparse
import asyncio
import base64
import json
import os

from cryptography.hazmat.primitives.asymmetric import ec
from mcp import Client


def _base64url_uint(value: int) -> str:
    """Encode a P-256 coordinate. / P-256座標をBase64URL形式へ変換する。"""
    raw = value.to_bytes(32, "big")
    return base64.urlsafe_b64encode(raw).rstrip(b"=").decode("ascii")


class MockClientAgent:
    """Minimal replaceable client. / 差し替え可能な最小Client。"""

    def __init__(
        self,
        downstream_mcp_url: str | None = None,
        access_token: str | None = None,
    ) -> None:
        self._private_key = ec.generate_private_key(ec.SECP256R1())

        self.downstream_mcp_url = downstream_mcp_url or os.getenv(
            "DOWNSTREAM_MCP_URL",
            "http://127.0.0.1:9000/mcp",
        )

        self.access_token = (
            access_token if access_token is not None else os.getenv("TEST_ACCESS_TOKEN")
        )

    def public_jwk(self) -> dict[str, str]:
        """Return the client's public JWK. / ClientのPublic JWKを返す。"""
        numbers = self._private_key.public_key().public_numbers()

        return {
            "kty": "EC",
            "crv": "P-256",
            "x": _base64url_uint(numbers.x),
            "y": _base64url_uint(numbers.y),
        }

    async def call_tool(self, tool: str, arguments: dict[str, object]):
        """Call an MCP tool. / MCP Toolを呼び出す。"""
        async with Client(self.downstream_mcp_url) as client:
            return await client.call_tool(tool, arguments)


async def main() -> None:
    parser = argparse.ArgumentParser(description="PoP mock MCP client")
    parser.add_argument("--url", default=None)

    subcommands = parser.add_subparsers(dest="command", required=True)

    subcommands.add_parser("key")

    create = subcommands.add_parser("create")
    create.add_argument("meeting_id")
    create.add_argument("title")

    delete = subcommands.add_parser("delete")
    delete.add_argument("meeting_id")

    args = parser.parse_args()

    agent = MockClientAgent(downstream_mcp_url=args.url)

    if args.command == "key":
        print(json.dumps(agent.public_jwk(), indent=2))
        return

    if args.command == "create":
        result = await agent.call_tool(
            "create_meeting",
            {
                "meeting_id": args.meeting_id,
                "title": args.title,
            },
        )
    else:
        result = await agent.call_tool(
            "delete_meeting",
            {
                "meeting_id": args.meeting_id,
            },
        )

    print(
        json.dumps(
            result.structured_content,
            indent=2,
            ensure_ascii=False,
        )
    )


if __name__ == "__main__":
    asyncio.run(main())
