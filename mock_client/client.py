"""Mock client agent for development. / 開発用Mock Client Agent。"""

from __future__ import annotations

import argparse
import asyncio
import base64
import hashlib
import json
import os
import time
import uuid
from urllib.parse import urlsplit, urlunsplit

import httpx2
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives.asymmetric.utils import decode_dss_signature
from mcp import Client
from mcp.client.streamable_http import streamable_http_client


def _base64url(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode("ascii")


def _base64url_uint(value: int) -> str:
    raw = value.to_bytes(32, "big")
    return _base64url(raw)


def _public_jwk(
    private_key: ec.EllipticCurvePrivateKey,
) -> dict[str, str]:
    numbers = private_key.public_key().public_numbers()

    return {
        "kty": "EC",
        "crv": "P-256",
        "x": _base64url_uint(numbers.x),
        "y": _base64url_uint(numbers.y),
    }


def _normalize_htu(url: str) -> str:
    parts = urlsplit(url)

    return urlunsplit(
        (
            parts.scheme,
            parts.netloc,
            parts.path,
            "",
            "",
        )
    )


def _create_dpop_proof(
    private_key: ec.EllipticCurvePrivateKey,
    access_token: str,
    method: str,
    url: str,
) -> str:
    header = {
        "typ": "dpop+jwt",
        "alg": "ES256",
        "jwk": _public_jwk(private_key),
    }

    payload = {
        "jti": str(uuid.uuid4()),
        "htm": method.upper(),
        "htu": _normalize_htu(url),
        "iat": int(time.time()),
        "ath": _base64url(hashlib.sha256(access_token.encode("ascii")).digest()),
    }

    encoded_header = _base64url(
        json.dumps(
            header,
            separators=(",", ":"),
            sort_keys=True,
        ).encode("utf-8")
    )

    encoded_payload = _base64url(
        json.dumps(
            payload,
            separators=(",", ":"),
            sort_keys=True,
        ).encode("utf-8")
    )

    signing_input = f"{encoded_header}.{encoded_payload}".encode("ascii")

    der_signature = private_key.sign(
        signing_input,
        ec.ECDSA(hashes.SHA256()),
    )

    r, s = decode_dss_signature(der_signature)

    signature = r.to_bytes(32, "big") + s.to_bytes(32, "big")

    return f"{encoded_header}.{encoded_payload}.{_base64url(signature)}"


def _load_private_key(
    path: str,
) -> ec.EllipticCurvePrivateKey:
    with open(path, "rb") as key_file:
        key = serialization.load_pem_private_key(
            key_file.read(),
            password=None,
        )

    if not isinstance(
        key,
        ec.EllipticCurvePrivateKey,
    ):
        raise ValueError("DPoP private key must be EC")

    if not isinstance(key.curve, ec.SECP256R1):
        raise ValueError("DPoP private key must use P-256")

    return key


class DPoPAuth(httpx2.Auth):
    """Create a fresh DPoP proof for every HTTP request."""

    def __init__(
        self,
        access_token: str,
        private_key: ec.EllipticCurvePrivateKey,
    ) -> None:
        self._access_token = access_token
        self._private_key = private_key

    def auth_flow(self, request):
        request.headers["Authorization"] = f"DPoP {self._access_token}"

        request.headers["DPoP"] = _create_dpop_proof(
            self._private_key,
            self._access_token,
            request.method,
            str(request.url),
        )

        yield request


class MockClientAgent:
    """Minimal replaceable client. / 差し替え可能な最小Client。"""

    def __init__(
        self,
        downstream_mcp_url: str | None = None,
        access_token: str | None = None,
        dpop_private_key_file: str | None = None,
    ) -> None:
        self.downstream_mcp_url = downstream_mcp_url or os.getenv(
            "DOWNSTREAM_MCP_URL",
            "http://127.0.0.1:9000/mcp",
        )

        self.access_token = (
            access_token if access_token is not None else os.getenv("TEST_ACCESS_TOKEN")
        )

        key_file = dpop_private_key_file or os.getenv("DPOP_PRIVATE_KEY_FILE")

        if key_file:
            self._private_key = _load_private_key(key_file)
        else:
            self._private_key = ec.generate_private_key(ec.SECP256R1())

    def public_jwk(self) -> dict[str, str]:
        """Return the client's public JWK."""
        return _public_jwk(self._private_key)

    async def call_tool(
        self,
        tool: str,
        arguments: dict[str, object],
    ):
        """Call an MCP tool. / MCP Toolを呼び出す。"""
        auth = None

        if self.access_token:
            auth = DPoPAuth(
                self.access_token,
                self._private_key,
            )

        async with httpx2.AsyncClient(
            auth=auth,
            timeout=httpx2.Timeout(
                30.0,
                read=300.0,
            ),
        ) as http_client:
            transport = streamable_http_client(
                self.downstream_mcp_url,
                http_client=http_client,
            )

            async with Client(transport) as client:
                return await client.call_tool(
                    tool,
                    arguments,
                )


async def main() -> None:
    parser = argparse.ArgumentParser(description="PoP mock MCP client")
    parser.add_argument("--url", default=None)

    subcommands = parser.add_subparsers(
        dest="command",
        required=True,
    )

    subcommands.add_parser("key")

    create = subcommands.add_parser("create")
    create.add_argument("meeting_id")
    create.add_argument("title")

    delete = subcommands.add_parser("delete")
    delete.add_argument("meeting_id")

    args = parser.parse_args()

    agent = MockClientAgent(downstream_mcp_url=args.url)

    if args.command == "key":
        print(
            json.dumps(
                agent.public_jwk(),
                indent=2,
            )
        )
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
