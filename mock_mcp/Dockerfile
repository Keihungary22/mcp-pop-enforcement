FROM python:3.13-slim

WORKDIR /app

COPY pyproject.toml ./
COPY mock_mcp ./mock_mcp
COPY mock_client ./mock_client

RUN python -m pip install --no-cache-dir .

ENV MOCK_MCP_SOCKET=/run/mcp/backend.sock

CMD ["python", "-m", "mock_mcp.server"]
