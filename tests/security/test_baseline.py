import base64
import json
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

import pytest
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives.asymmetric.utils import decode_dss_signature

from mock_client.client import _create_dpop_proof, _public_jwk

TOKEN_URL = "http://host.docker.internal:8080/realms/mcp-pop/protocol/openid-connect/token"

RESOURCE_URL = "http://127.0.0.1:9100/mcp"

CLIENT_ID = "mcp-client"
CLIENT_SECRET = "mcp-client-secret"


def _base64url(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()


def _create_token_endpoint_proof(
    private_key: ec.EllipticCurvePrivateKey,
) -> str:
    header = {
        "typ": "dpop+jwt",
        "alg": "ES256",
        "jwk": _public_jwk(private_key),
    }

    payload = {
        "jti": str(uuid.uuid4()),
        "htm": "POST",
        "htu": TOKEN_URL,
        "iat": int(time.time()),
    }

    encoded_header = _base64url(
        json.dumps(
            header,
            separators=(",", ":"),
            sort_keys=True,
        ).encode()
    )

    encoded_payload = _base64url(
        json.dumps(
            payload,
            separators=(",", ":"),
            sort_keys=True,
        ).encode()
    )

    signing_input = (f"{encoded_header}.{encoded_payload}").encode()

    der_signature = private_key.sign(
        signing_input,
        ec.ECDSA(hashes.SHA256()),
    )

    r, s = decode_dss_signature(der_signature)

    signature = r.to_bytes(32, "big") + s.to_bytes(32, "big")

    return f"{encoded_header}.{encoded_payload}.{_base64url(signature)}"


def _request_access_token(
    private_key: ec.EllipticCurvePrivateKey,
) -> str:
    proof = _create_token_endpoint_proof(
        private_key,
    )

    data = urllib.parse.urlencode(
        {
            "grant_type": "client_credentials",
            "client_id": CLIENT_ID,
            "client_secret": CLIENT_SECRET,
            "scope": "mcp:invoke",
        }
    ).encode()

    request = urllib.request.Request(
        TOKEN_URL,
        data=data,
        method="POST",
        headers={
            "Content-Type": "application/x-www-form-urlencoded",
            "DPoP": proof,
        },
    )

    with urllib.request.urlopen(
        request,
        timeout=10,
    ) as response:
        assert response.status == 200

        body = json.loads(response.read().decode())

    assert body["token_type"] == "DPoP"

    return body["access_token"]


def _mcp_body() -> bytes:
    return json.dumps(
        {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2025-11-25",
                "capabilities": {},
                "clientInfo": {
                    "name": "security-baseline",
                    "version": "1.0",
                },
            },
        }
    ).encode()


def _send_mcp(
    token: str,
    proof: str | None,
) -> tuple[int, str]:
    headers = {
        "Authorization": f"DPoP {token}",
        "Content-Type": "application/json",
        "Accept": "application/json, text/event-stream",
    }

    if proof is not None:
        headers["DPoP"] = proof

    request = urllib.request.Request(
        RESOURCE_URL,
        data=_mcp_body(),
        method="POST",
        headers=headers,
    )

    try:
        with urllib.request.urlopen(
            request,
            timeout=10,
        ) as response:
            return (
                response.status,
                response.read().decode(),
            )

    except urllib.error.HTTPError as error:
        return (
            error.code,
            error.read().decode(),
        )


@pytest.fixture
def client_credentials():
    private_key = ec.generate_private_key(ec.SECP256R1())

    token = _request_access_token(
        private_key,
    )

    return private_key, token


def test_valid_dpop_bound_request_is_allowed(
    client_credentials,
):
    private_key, token = client_credentials

    proof = _create_dpop_proof(
        private_key,
        token,
        "POST",
        RESOURCE_URL,
    )

    status, body = _send_mcp(
        token,
        proof,
    )

    assert status == 200
    assert '"result"' in body


def test_stolen_token_without_proof_is_rejected(
    client_credentials,
):
    _, token = client_credentials

    status, _ = _send_mcp(
        token,
        None,
    )

    assert status == 401


def test_stolen_token_with_unrelated_key_is_rejected(
    client_credentials,
):
    _, token = client_credentials

    attacker_key = ec.generate_private_key(ec.SECP256R1())

    attacker_proof = _create_dpop_proof(
        attacker_key,
        token,
        "POST",
        RESOURCE_URL,
    )

    status, body = _send_mcp(
        token,
        attacker_proof,
    )

    assert status == 401
    assert "binding mismatch" in body.lower()


def test_replayed_dpop_proof_is_rejected(
    client_credentials,
):
    private_key, token = client_credentials

    proof = _create_dpop_proof(
        private_key,
        token,
        "POST",
        RESOURCE_URL,
    )

    first_status, _ = _send_mcp(
        token,
        proof,
    )

    second_status, second_body = _send_mcp(
        token,
        proof,
    )

    assert first_status == 200
    assert second_status == 401
    assert "replay" in second_body.lower()


def test_wrong_http_method_in_proof_is_rejected(
    client_credentials,
):
    private_key, token = client_credentials

    proof = _create_dpop_proof(
        private_key,
        token,
        "GET",
        RESOURCE_URL,
    )

    status, _ = _send_mcp(
        token,
        proof,
    )

    assert status == 401


def test_forwarded_proof_for_another_resource_is_rejected(
    client_credentials,
):
    private_key, token = client_credentials

    proof = _create_dpop_proof(
        private_key,
        token,
        "POST",
        "http://127.0.0.1:9200/mcp",
    )

    status, _ = _send_mcp(
        token,
        proof,
    )

    assert status == 401


def _request_limited_access_token(
    private_key: ec.EllipticCurvePrivateKey,
) -> str:
    proof = _create_token_endpoint_proof(private_key)

    data = urllib.parse.urlencode(
        {
            "grant_type": "client_credentials",
            "client_id": "mcp-client-limited",
            "client_secret": "mcp-client-limited-secret",
        }
    ).encode()

    request = urllib.request.Request(
        TOKEN_URL,
        data=data,
        method="POST",
        headers={
            "Content-Type": "application/x-www-form-urlencoded",
            "DPoP": proof,
        },
    )

    with urllib.request.urlopen(
        request,
        timeout=10,
    ) as response:
        body = json.loads(response.read().decode())

    assert body["token_type"] == "DPoP"

    return body["access_token"]


def _decode_jwt_payload(token: str) -> dict:
    payload = token.split(".")[1]
    padding = "=" * (-len(payload) % 4)

    return json.loads(base64.urlsafe_b64decode(payload + padding))


def test_insufficient_scope_is_rejected():
    private_key = ec.generate_private_key(ec.SECP256R1())

    token = _request_limited_access_token(
        private_key,
    )

    claims = _decode_jwt_payload(token)

    assert (
        "mcp:invoke"
        not in claims.get(
            "scope",
            "",
        ).split()
    )

    proof = _create_dpop_proof(
        private_key,
        token,
        "POST",
        RESOURCE_URL,
    )

    status, _ = _send_mcp(
        token,
        proof,
    )

    assert status == 403


def test_expired_access_token_is_rejected(
    client_credentials,
):
    private_key, token = client_credentials

    claims = _decode_jwt_payload(token)

    remaining = claims["exp"] - int(time.time())

    if remaining >= 0:
        time.sleep(remaining + 1)

    proof = _create_dpop_proof(
        private_key,
        token,
        "POST",
        RESOURCE_URL,
    )

    status, _ = _send_mcp(
        token,
        proof,
    )

    assert status == 401


def test_revoked_access_token_is_rejected():
    private_key = ec.generate_private_key(ec.SECP256R1())

    token = _request_access_token(
        private_key,
    )

    claims = _decode_jwt_payload(token)

    before_proof = _create_dpop_proof(
        private_key,
        token,
        "POST",
        RESOURCE_URL,
    )

    before_status, _ = _send_mcp(
        token,
        before_proof,
    )

    assert before_status == 200

    revocation_url = (
        "http://host.docker.internal:8080/realms/mcp-pop/protocol/openid-connect/revoke"
    )

    data = urllib.parse.urlencode(
        {
            "token": token,
            "token_type_hint": "access_token",
            "client_id": CLIENT_ID,
            "client_secret": CLIENT_SECRET,
        }
    ).encode()

    request = urllib.request.Request(
        revocation_url,
        data=data,
        method="POST",
        headers={
            "Content-Type": "application/x-www-form-urlencoded",
        },
    )

    with urllib.request.urlopen(
        request,
        timeout=10,
    ) as response:
        assert response.status == 200

    # Prove that rejection is caused by revocation,
    # not ordinary token expiration.
    #
    # 通常の期限切れではなく、失効による拒否であることを確認する。
    assert claims["exp"] > int(time.time())

    after_proof = _create_dpop_proof(
        private_key,
        token,
        "POST",
        RESOURCE_URL,
    )

    after_status, _ = _send_mcp(
        token,
        after_proof,
    )

    assert after_status == 401


def _request_other_resource_access_token(
    private_key: ec.EllipticCurvePrivateKey,
) -> str:
    proof = _create_token_endpoint_proof(
        private_key,
    )

    data = urllib.parse.urlencode(
        {
            "grant_type": "client_credentials",
            "client_id": "mcp-client-other",
            "client_secret": "mcp-client-other-secret",
            "scope": "mcp:invoke",
        }
    ).encode()

    request = urllib.request.Request(
        TOKEN_URL,
        data=data,
        method="POST",
        headers={
            "Content-Type": "application/x-www-form-urlencoded",
            "DPoP": proof,
        },
    )

    with urllib.request.urlopen(
        request,
        timeout=10,
    ) as response:
        body = json.loads(response.read().decode())

    assert body["token_type"] == "DPoP"

    return body["access_token"]


def test_cross_resource_token_substitution_is_rejected():
    """Reject a valid token issued for another resource.

    別Resource向けに正規発行されたTokenの流用を拒否する。
    """
    private_key = ec.generate_private_key(ec.SECP256R1())

    token = _request_other_resource_access_token(
        private_key,
    )

    claims = _decode_jwt_payload(token)

    assert claims["aud"] == "other-resource"
    assert (
        "mcp:invoke"
        in claims.get(
            "scope",
            "",
        ).split()
    )
    assert claims.get("cnf", {}).get("jkt")

    proof = _create_dpop_proof(
        private_key,
        token,
        "POST",
        RESOURCE_URL,
    )

    status, _ = _send_mcp(
        token,
        proof,
    )

    assert status == 401
