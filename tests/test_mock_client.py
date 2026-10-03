from mock_client.client import MockClientAgent


def test_public_jwk_is_p256() -> None:
    agent = MockClientAgent()

    jwk = agent.public_jwk()

    assert jwk["kty"] == "EC"
    assert jwk["crv"] == "P-256"
    assert jwk["x"]
    assert jwk["y"]


def test_downstream_url_from_environment(monkeypatch) -> None:
    monkeypatch.setenv(
        "DOWNSTREAM_MCP_URL",
        "http://example.test:9100/mcp",
    )

    agent = MockClientAgent()

    assert agent.downstream_mcp_url == "http://example.test:9100/mcp"
