FROM python:3.13-slim

WORKDIR /app

COPY pyproject.toml ./
COPY mock_mcp ./mock_mcp
COPY mock_client ./mock_client

RUN python -m pip install --no-cache-dir .

ENV MOCK_MCP_HOST=0.0.0.0
ENV MOCK_MCP_PORT=9000

EXPOSE 9000

CMD ["python", "-m", "mock_mcp.server"]
