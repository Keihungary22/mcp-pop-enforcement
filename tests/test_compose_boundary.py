import json
import subprocess


def test_mock_mcp_has_no_published_host_ports():
    result = subprocess.run(
        ["docker", "compose", "config", "--format", "json"],
        capture_output=True,
        text=True,
        check=True,
    )

    config = json.loads(result.stdout)
    services = config["services"]

    assert not services["mock-mcp"].get("ports")
    assert services["pop-enforcement"].get("ports")
