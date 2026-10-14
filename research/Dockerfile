FROM python:3.12-slim

WORKDIR /app
COPY pyproject.toml ./
COPY mock_mcp/ ./mock_mcp/
COPY mock_client/ ./mock_client/
COPY research/ ./research/

RUN pip install --no-cache-dir -e .

ENV RESEARCH_MCP_HOST=0.0.0.0
ENV RESEARCH_MCP_PORT=9000

CMD ["python", "-m", "research.calendar_lab.server"]
