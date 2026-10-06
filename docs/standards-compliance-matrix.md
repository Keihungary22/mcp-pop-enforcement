# Standards Compliance Matrix — Baseline v1

## 1. Purpose

This document records the standards requirements evaluated for the
`baseline-v1` reference baseline.

Baseline snapshot:

- Git tag: `baseline-v1`
- Commit: `8b4f41b29a5cd0aaf361879922a6c210e160f538`

The purpose of this matrix is traceability:

RFC requirement -> implementation -> test/evidence -> status

This document does not claim complete conformance to every requirement
contained in the referenced RFCs. Requirements are evaluated only where
they are applicable to the roles and mechanisms selected for the baseline.

## 2. Baseline Roles and Standards Scope

The experimental system contains the following relevant roles:

- Keycloak: OAuth Authorization Server
- Python mock client / security tests: OAuth client
- Go enforcement gateway: OAuth Protected Resource / Resource Server
- Mock MCP server: downstream protected application

The primary standards target is RFC 9449 for DPoP-based protected-resource
enforcement.

RFC 7638 is used for JWK thumbprint calculation.

RFC 7662 is used for token activity and revocation-state checking.

RFC 7009 is exercised by the revocation security test, but complete
RFC 7009 conformance is not claimed.

RFC 9068 was evaluated but is not a conformance target of Baseline v1.

## 3. Status Definitions

- `VERIFIED`: Applicable behavior is present and supported by implementation
  and/or direct test evidence.
- `IMPLEMENTED-NOT-DIRECTLY-TESTED`: Implementation evidence exists, but no
  dedicated test was identified during this audit.
- `ENVIRONMENT-DEVIATION`: The requirement is intentionally not satisfied by
  the local experimental environment.
- `NOT-APPLICABLE`: The conditional requirement does not apply to the current
  baseline configuration.
- `NOT-A-TARGET`: The RFC/profile was reviewed but was not selected as a
  conformance target.
- `NOT-ASSESSED`: Evidence was insufficient to make a claim.

## 4. Compliance Matrix

| Specification | Section / mechanism | Applicable requirement | Component | Evidence | Status | Notes |
|---|---|---|---|---|---|---|
| RFC 9449 | DPoP proof JWT | DPoP proof uses `typ=dpop+jwt` | Go gateway | `internal/dpop/proof.go`; wrong-`typ` unit test | VERIFIED | Other `typ` values are rejected |
| RFC 9449 | DPoP proof JWT | Proof uses an asymmetric signing algorithm and is signed by the key represented by `jwk` | Go gateway | ES256-only parser and signature verification | VERIFIED | Baseline intentionally restricts proofs to ES256 |
| RFC 9449 | DPoP proof JWT | `jwk` contains the public key and must not contain private-key material | Go gateway | `internal/dpop/jwk.go`; private-key-material rejection test | VERIFIED | EC P-256 is used |
| RFC 9449 | DPoP claims | `jti`, `htm`, `htu`, and `iat` are present and validated | Go gateway | `internal/dpop/proof.go`; missing/wrong/stale/future tests | VERIFIED | Method and target URI are request-bound |
| RFC 9449 | Access-token binding | Protected-resource proof contains valid `ath` | Go gateway | `AccessTokenHash`; missing/wrong `ath` tests | VERIFIED | SHA-256 hash binds proof to token value |
| RFC 9449 | Public-key confirmation | DPoP proof key matches the key bound to the access token | Go gateway | `cnf.jkt` comparison in enforcement middleware | VERIFIED | Prevents use with an unrelated DPoP key |
| RFC 9449 | Protected-resource access | DPoP-bound token is presented using the `DPoP` authorization scheme together with a DPoP proof | Go gateway | Enforcement middleware; Bearer-rejection test | VERIFIED | Bearer downgrade is rejected |
| RFC 9449 | Proof freshness | Proof creation time must be within an acceptable time window | Go gateway | `iat` validation; stale/future proof tests | VERIFIED | Exact window is baseline configuration |
| RFC 9449 | Replay mitigation | Previously accepted proof identifiers are detected within the replay window | Go gateway | `MemoryReplayStore`; replay unit and integration tests | VERIFIED | `jkt:jti` is stored; this is an additional replay-control mechanism |
| RFC 9449 | Nonce | A nonce must be validated when the server has supplied one | Go gateway | No DPoP nonce is issued by the baseline | NOT-APPLICABLE | Nonce support is not enabled |
| RFC 9449 | Secure transport | DPoP must be used together with HTTPS | Experimental environment | Local endpoints use HTTP | ENVIRONMENT-DEVIATION | Acceptable only as an isolated development/test setup; production deployment requires HTTPS |
| RFC 9449 / RFC 7638 | JWK thumbprint (`cnf.jkt`) | `jkt` uses the RFC 7638 JWK thumbprint construction with SHA-256 as required by RFC 9449 §6.1 | Go gateway | `JWK.Thumbprint()` and deterministic thumbprint unit test | VERIFIED | Used for DPoP key binding |
| RFC 7662 | Introspection request | Introspection is performed using HTTP POST and form-encoded parameters | Go gateway | `internal/auth/introspection.go` | VERIFIED | `token` parameter is supplied |
| RFC 7662 | Endpoint authorization | Introspection endpoint requires authorization | Keycloak / Go gateway | HTTP Basic authentication in client; unauthenticated request returned HTTP 401 | VERIFIED | Verified against running Keycloak instance |
| RFC 7662 | Introspection response | `active` result is used to determine token activity | Go gateway | Active/inactive unit tests | VERIFIED | Inactive tokens are rejected |
| RFC 7662 | `token_type_hint` | Optional token-type hint | Go gateway | Not sent | NOT-APPLICABLE | The parameter is optional |
| RFC 7662 | Secure transport | Introspection endpoint is protected by transport-layer security | Experimental environment | Local Keycloak endpoint uses HTTP | ENVIRONMENT-DEVIATION | Production deployment requires TLS |
| RFC 7009 | Access-token revocation | Authorization server supports access-token revocation | Keycloak / security tests | Revoked-but-unexpired token integration scenario | VERIFIED | Only access-token behavior relevant to this experiment was evaluated |
| RFC 7009 | Secure transport | Published revocation endpoint uses HTTPS | Experimental environment | Local Keycloak endpoint uses HTTP | ENVIRONMENT-DEVIATION | Complete RFC 7009 conformance is not claimed |
| RFC 9068 | JWT Access Token Profile | JWT access token uses the RFC 9068 profile | Keycloak / Go gateway | Runtime token had `typ=JWT`, lacked `client_id`; gateway does not enforce `at+jwt` | NOT-A-TARGET | Baseline v1 does not claim RFC 9068 conformance |

## 5. RFC 9068 Decision

RFC 9068 was explicitly evaluated during the baseline audit.

The runtime Keycloak access token contained:

- `iss`
- `exp`
- `aud`
- `sub`
- `iat`
- `jti`
- `scope`
- `cnf`

However:

- its JWT `typ` header was `JWT`, not `at+jwt`; and
- the token did not contain `client_id`; and
- the resource-server validator does not enforce the RFC 9068 access-token
  `typ`.

Therefore Baseline v1 does not claim conformance to the RFC 9068 JWT Access
Token Profile.

This does not prevent the baseline from using OAuth access tokens with DPoP.
RFC 9068 defines a particular interoperable JWT access-token profile and is
not a prerequisite for use of DPoP.

## 6. Experimental Transport Deviation

The baseline runs entirely in an isolated local Docker-based experimental
environment.

HTTP is currently used for the Keycloak, gateway, and MCP endpoints to keep
the experiment reproducible and locally observable.

This creates deliberate transport-level deviations from specifications that
require secure transport, including RFC 9449, RFC 7662, and RFC 7009.

These deviations are environmental rather than changes to the evaluated
DPoP, token-introspection, or delegation-enforcement logic. Any production
deployment would require HTTPS/TLS.

## 7. Interpretation for the Research

The standards audit establishes a standards-based reference point without
claiming universal OAuth compliance.

In particular, Baseline v1 enforces:

- access-token signature and validity checks;
- expected issuer and audience;
- required scope;
- DPoP proof validation;
- proof-to-request binding;
- proof-to-token binding;
- token-to-key binding;
- replay detection; and
- authorization-server token activity checking.

The subsequent research therefore focuses on a different question:

> Even when token validity and proof-of-possession are enforced, what
> delegation semantics remain unenforced for agent-to-agent actions?

That question is addressed separately by the threat model and the proposed
delegation-aware extension.

## 8. Referenced Standards

- RFC 9449 — OAuth 2.0 Demonstrating Proof of Possession (DPoP)
- RFC 7638 — JSON Web Key (JWK) Thumbprint
- RFC 7662 — OAuth 2.0 Token Introspection
- RFC 7009 — OAuth 2.0 Token Revocation
- RFC 9068 — JSON Web Token (JWT) Profile for OAuth 2.0 Access Tokens
