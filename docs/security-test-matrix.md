# Security Test Matrix

This document summarizes the baseline security scenarios validated by the
PoP enforcement prototype.

The protected-resource flow is:

```text
Client
  |
  | DPoP-bound access token
  | DPoP proof
  v
Go Enforcement Gateway
  |
  +-- JWT / JWKS validation
  +-- issuer validation
  +-- audience validation
  +-- scope validation
  +-- expiration validation
  +-- token introspection / revocation check
  +-- DPoP proof validation
  +-- cnf.jkt key binding
  +-- replay protection
  |
  v
Mock MCP Service
```

## Baseline Security Scenarios

| Scenario | Expected Result | Enforcement Mechanism |
|---|---|---|
| Valid DPoP-bound request | Allow | Full validation chain |
| Stolen token without DPoP proof | Reject (401) | DPoP proof requirement |
| Stolen token with unrelated key | Reject (401) | `cnf.jkt` key binding |
| Replayed DPoP proof | Reject (401) | JKT + JTI replay cache |
| Wrong HTTP method in proof | Reject (401) | `htm` validation |
| Proof forwarded to another resource URI | Reject (401) | `htu` validation |
| Insufficient scope / privilege escalation | Reject (403) | Required scope validation |
| Expired access token | Reject (401) | `exp` validation |
| Revoked access token | Reject (401) | Authorization-server introspection |
| Token issued for another resource | Reject (401) | `aud` validation |

## Revocation Finding

The original JWKS-only baseline could validate JWT signatures and claims,
but it could not immediately observe authorization-server revocation state.

Experimentally:

```text
JWKS-only baseline

Fresh token          -> 200
Revoke token         -> 200
Same unexpired token -> 200
```

The enforcement layer was therefore extended with token introspection.

After the mitigation:

```text
JWKS + Introspection

Fresh token          -> 200
Revoke token         -> 200
Same unexpired token -> 401
```

The revocation test additionally verifies that the JWT has not yet expired,
so rejection cannot be explained by ordinary expiration.

## Cross-Resource Misuse

A separate Keycloak resource and client are used to obtain a legitimate
DPoP-bound token with:

```text
aud   = other-resource
scope = mcp:invoke
cnf.jkt = valid client key thumbprint
```

The same token and its correct possession key are then presented to the
`mcp-resource` enforcement gateway.

The gateway rejects the request because the token audience does not match
the protected resource.

This is treated as a confused-deputy-style cross-resource misuse baseline.
It does not claim to solve the general agent-delegation confused-deputy
problem. Task binding, delegation context, and delegation-chain semantics
remain possible research extensions.

## Local Development Note

The local Docker Desktop environment uses:

```text
http://host.docker.internal:8080
```

as the canonical Keycloak issuer so that the test client and Dockerized
enforcement gateway observe the same issuer identity.

This hostname is a local-development detail. A deployed environment should
use its actual stable authorization-server URL.

＝＝＝＝＝＝＝＝＝＝＝＝＝＝＝

# セキュリティテストマトリクス

この文書は、PoP Enforcement Prototypeで検証したBaseline Security
Scenarioをまとめたものです。

Protected ResourceへのRequestは次の流れで検証されます。

```text
Client
  |
  | DPoP-bound Access Token
  | DPoP Proof
  v
Go Enforcement Gateway
  |
  +-- JWT / JWKS検証
  +-- issuer検証
  +-- audience検証
  +-- scope検証
  +-- expiration検証
  +-- Token Introspection / Revocation確認
  +-- DPoP Proof検証
  +-- cnf.jkt Key Binding
  +-- Replay Protection
  |
  v
Mock MCP Service
```

## Baseline Security Scenario

| Scenario | 期待結果 | Enforcement Mechanism |
|---|---|---|
| 正常なDPoP-bound Request | 許可 | 全検証チェーン |
| DPoP Proofなしの盗難Token | 401 | DPoP Proof必須 |
| 別Keyで盗難Tokenを使用 | 401 | `cnf.jkt` Key Binding |
| DPoP Proof Replay | 401 | JKT + JTI Replay Cache |
| ProofのHTTP Method不一致 | 401 | `htm`検証 |
| 別Resource URIへのProof Forwarding | 401 | `htu`検証 |
| Scope不足・Privilege Escalation | 403 | Required Scope検証 |
| Expired Token | 401 | `exp`検証 |
| Revoked Token | 401 | Authorization Server Introspection |
| 別Resource向けTokenの流用 | 401 | `aud`検証 |

## Revocationで確認したこと

当初のJWKS-only構成では、JWTの署名やClaimは検証できても、
Authorization Server側の即時Revocation Stateを知ることができませんでした。

実験では、

```text
JWKS-only baseline

Fresh Token          -> 200
TokenをRevoke        -> 200
期限内の同じToken    -> 200
```

となりました。

そこでEnforcement LayerへToken Introspectionを追加しました。

対策後は、

```text
JWKS + Introspection

Fresh Token          -> 200
TokenをRevoke        -> 200
期限内の同じToken    -> 401
```

となります。

Revocation TestではTokenの`exp`がまだ未来であることも確認しているため、
401が単なるToken Expirationによるものではないことも検証しています。

## Cross-Resource Misuse

別のKeycloak ResourceとClientから、

```text
aud   = other-resource
scope = mcp:invoke
cnf.jkt = 正しいClient Key Thumbprint
```

を持つ正規のDPoP-bound Tokenを発行します。

そのTokenと正しい秘密鍵を`mcp-resource` Gatewayへ流用しても、
Audienceが一致しないためRequestは拒否されます。

これはConfused Deputyに関連するCross-Resource MisuseのBaseline Testとして
扱います。

ただし、一般的なAgent DelegationにおけるConfused Deputy問題を完全に
解決したことを意味するものではありません。

Task Binding、Delegation Context、Delegation Chainなどは今後の研究拡張候補です。

## Local Development Note

Docker Desktop上ではClientとDockerized Gatewayが同じIssuer Identityを
参照できるよう、

```text
http://host.docker.internal:8080
```

をKeycloakのCanonical Issuerとして使用しています。

これはLocal Development用の設定であり、本番環境では実際の固定された
Authorization Server URLを使用します。
