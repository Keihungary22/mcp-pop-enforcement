# Standalone PoP Prototype Interface Specification

## 1. Purpose

This document defines the external interfaces that the standalone
Proof-of-Possession prototype must preserve for later integration with the
shared Secure-agenticAI repository.

The standalone implementation may differ internally, but its externally
observable behavior should remain compatible with the shared system.

No source code from the shared repository is copied into this prototype.

## 2. Standalone Architecture

Default local topology:

```text
Mock Client
  localhost:8002
        |
        v
PoP Enforcement Service
  localhost:9100
        |
        v
Mock MCP Service
  localhost:9000
```

Default values are development defaults only and must be replaceable through
environment variables.

## 3. Agent B Compatibility

The later integration target exposes the following Agent B interfaces:

### Agent card

```http
GET /.well-known/agent-card.json
```

The standalone prototype does not need to implement the complete Agent B
application initially, but compatibility with this discovery path must be
preserved.

### A2A endpoint

```http
POST /a2a
```

Requests originating from the agent layer may eventually reach MCP resources
through the PoP enforcement layer.

## 4. MCP Interface

The MCP service endpoint is:

```http
/mcp
```

Standalone default:

```text
http://localhost:9000/mcp
```

The enforcement service forwards an authorized request to this upstream MCP
endpoint only after OAuth access-token and DPoP validation succeeds.

## 5. Authorization Headers

Protected requests use DPoP-bound OAuth access tokens.

Required headers:

```http
Authorization: DPoP <access-token>
DPoP: <proof>
```

The `Authorization` scheme must be `DPoP`, not `Bearer`, for requests protected
by this prototype.

## 6. Access Token Claims

The prototype must be able to validate or consume the following relevant
claims:

- `iss` - token issuer
- `sub` - subject
- `aud` - intended audience
- `scope` - delegated permissions
- `iat` - issued-at time
- `exp` - expiration time
- `cnf.jkt` - thumbprint of the public key to which the token is bound

Additional resource or delegation constraints may be added later as the
research prototype evolves.

The critical PoP property is that `cnf.jkt` matches the public key represented
by the DPoP proof.

## 7. DPoP Proof Requirements

The `DPoP` header contains a signed JWT proof.

Relevant JWT header fields include:

- `typ`
- `alg`
- `jwk`

Relevant proof claims include:

- `htu` - target URI
- `htm` - HTTP method
- `iat` - issued-at time
- `jti` - unique proof identifier
- `ath` - access-token hash when required

The enforcement service must verify that:

1. the proof signature is valid;
2. the public key is acceptable;
3. the proof key matches the token `cnf.jkt`;
4. `htm` matches the actual HTTP method;
5. `htu` matches the protected request URI;
6. `iat` is within the accepted time window;
7. `jti` has not already been used within the replay window;
8. `ath`, when required, matches the presented access token.

Replay detection is implemented in a later issue.

## 8. Service URLs and Ports

Development defaults:

| Component | Default |
| --- | --- |
| Mock Client | `http://localhost:8002` |
| PoP Enforcement | `http://localhost:9100` |
| Mock MCP | `http://localhost:9000` |
| MCP endpoint | `http://localhost:9000/mcp` |

These values are not part of the security model and must remain configurable.

## 9. Environment Configuration

At minimum, the standalone prototype should support configuration equivalent
to:

```text
ENFORCEMENT_HOST=0.0.0.0
ENFORCEMENT_PORT=9100

UPSTREAM_MCP_URL=http://localhost:9000/mcp

OAUTH_ISSUER_URL=<issuer>
OAUTH_AUDIENCE=<audience>

DPoP_IAT_WINDOW_SECONDS=<window>
DPoP_REPLAY_WINDOW_SECONDS=<window>
```

Keycloak-specific configuration will be introduced when the OAuth provider is
added.

Secrets, credentials, private keys, access tokens, and environment-specific
production values must not be committed to this repository.

## 10. Enforcement Flow

Expected request flow:

```text
Client
  |
  | Authorization: DPoP <access-token>
  | DPoP: <proof>
  v
PoP Enforcement Service
  |
  | 1. validate access token
  | 2. validate DPoP proof
  | 3. verify token/key binding
  | 4. enforce replay protection
  | 5. enforce delegated authorization
  v
MCP Service
```

The enforcement component acts as the security boundary between the requesting
agent/client and the MCP service.

## 11. Integration Constraints

The standalone prototype should preserve:

- Agent B discovery path
- Agent B A2A path
- MCP `/mcp` endpoint
- `Authorization: DPoP <access-token>`
- `DPoP: <proof>`
- relevant OAuth token claims
- DPoP proof semantics
- configurable service URLs and ports

Internal package structure, storage, test utilities, and implementation details
may differ from the shared repository.

## 12. Scope of This Issue

This issue defines the interface specification only.

Implementation of the following belongs to later issues:

- mock client and mock MCP service
- Go enforcement gateway
- OAuth token validation
- DPoP validation
- key binding
- replay protection
- Keycloak integration
- security and integration tests
