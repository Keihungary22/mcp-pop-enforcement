import http.client
import time
import urllib.error
import urllib.request

KEYCLOAK_DISCOVERY = (
    "http://host.docker.internal:8080/realms/mcp-pop/.well-known/openid-configuration"
)

ENFORCEMENT_HEALTH = "http://127.0.0.1:9100/healthz"


def wait_for_http(
    url: str,
    timeout: float = 150.0,
    interval: float = 2.0,
) -> bytes:
    """Wait until an HTTP endpoint becomes ready.

    HTTP Endpointが利用可能になるまで待機する。
    """
    deadline = time.monotonic() + timeout
    last_error = None

    while time.monotonic() < deadline:
        try:
            with urllib.request.urlopen(
                url,
                timeout=5,
            ) as response:
                if response.status == 200:
                    return response.read()

        except (
            urllib.error.URLError,
            urllib.error.HTTPError,
            http.client.RemoteDisconnected,
            TimeoutError,
            ConnectionError,
        ) as exc:
            last_error = exc

        time.sleep(interval)

    raise AssertionError(f"endpoint did not become ready: {url}; last error: {last_error}")


def test_keycloak_is_available():
    body = wait_for_http(KEYCLOAK_DISCOVERY)

    assert b'"issuer"' in body


def test_enforcement_is_available():
    body = wait_for_http(
        ENFORCEMENT_HEALTH,
        timeout=30.0,
    )

    assert body.decode() == "ok"
