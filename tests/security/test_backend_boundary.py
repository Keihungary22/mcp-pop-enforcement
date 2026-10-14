import json
import socket
import subprocess
import sys

import pytest


def docker(*args: str) -> str:
    result = subprocess.run(
        ["docker", *args],
        check=True,
        capture_output=True,
        text=True,
        timeout=30,
    )
    return result.stdout.strip()


def backend_container_id() -> str:
    return docker("compose", "ps", "-q", "mock-mcp")


def test_backend_tcp_port_is_closed() -> None:
    container = backend_container_id()
    assert container, "Mock MCP container is not running"

    code = (
        "import socket; "
        "s=socket.socket(); "
        "s.settimeout(3); "
        "result=s.connect_ex(('127.0.0.1',9000)); "
        "s.close(); "
        "assert result != 0, 'Backend TCP port is reachable'"
    )
    docker("exec", container, "python", "-c", code)


def test_socket_permissions() -> None:
    container = backend_container_id()
    assert container, "Mock MCP container is not running"

    code = (
        "import os,stat; "
        "s=os.stat('/run/mcp/backend.sock'); "
        "assert stat.S_ISSOCK(s.st_mode); "
        "assert s.st_gid == 101; "
        "assert stat.S_IMODE(s.st_mode) == 0o660"
    )
    docker("exec", container, "python", "-c", code)


@pytest.mark.skipif(
    sys.platform != "linux",
    reason="Host-to-container bridge test requires native Linux",
)
def test_linux_host_cannot_connect_to_backend_ip() -> None:
    container = backend_container_id()
    assert container, "Mock MCP container is not running"

    info = json.loads(docker("inspect", container))[0]
    networks = info["NetworkSettings"]["Networks"]
    addresses = [network["IPAddress"] for network in networks.values() if network.get("IPAddress")]
    assert addresses, "No container IPv4 address found"

    for address in addresses:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as connection:
            connection.settimeout(3)
            result = connection.connect_ex((address, 9000))
            assert result != 0, f"Gateway bypass possible via {address}:9000"
