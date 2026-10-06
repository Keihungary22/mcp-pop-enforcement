# Baseline v1 Definition

## 1. Purpose

Baseline v1 defines the reference security architecture used in this
research for comparison with a future delegation-aware enforcement
mechanism.

The baseline combines OAuth-based authorization, DPoP-based
proof-of-possession, access-token validation, replay protection, and
revocation checking.

It intentionally does not enforce explicit agent-to-agent delegation
semantics.

## 2. Baseline Boundary

Baseline v1 answers the following question:

> Is this access token valid, intended for this protected resource,
> sufficiently authorized, still active, and being used by the holder
> of the cryptographic key to which it is bound?

Baseline v1 does not answer:

> Was this agent explicitly delegated this exact action for this exact
> task by an authorized delegator?

This distinction defines the boundary between the reference baseline
and the proposed delegation-aware extension.

## 3. Included Mechanisms

Baseline v1 includes:

- Keycloak as the OAuth authorization server
- DPoP-bound access-token issuance
- JWT signature validation using JWKS
- issuer (`iss`) validation
- audience (`aud`) validation
- scope validation
- expiration (`exp`) validation
- DPoP proof validation
- token-to-key binding using `cnf.jkt`
- HTTP method (`htm`) validation
- target URI (`htu`) validation
- access-token hash (`ath`) validation
- DPoP replay detection
- token introspection
- revocation enforcement

## 4. Reference Architecture

~~~text
                    +----------------------+
                    |      Keycloak        |
                    | Authorization Server |
                    +----------+-----------+
                               |
                 token / JWKS / introspection
                               |
                               v
+------------+          +------+-----------+          +------------+
|   Client   | -------> | Go Enforcement   | -------> | MCP Server |
| DPoP Key   | token +  | Gateway          | validated|            |
+------------+ proof    +------------------+ request  +------------+
~~~

The enforcement gateway acts as the protected-resource security
boundary before requests reach the MCP server.

## 5. Explicit Exclusions

The following mechanisms are not part of Baseline v1:

- explicit delegator identity binding
- explicit delegatee identity binding
- task-specific authorization
- action-specific delegation constraints
- delegation context binding
- delegation-chain validation
- delegation-depth restrictions
- agent-to-agent authority propagation rules

These mechanisms are reserved for the proposed delegation-aware
research extension.

## 6. Acceptance Criteria

Baseline v1 is considered functionally valid when the following
security scenarios produce the expected result.

| Scenario | Expected Result |
|---|---|
| Valid DPoP-bound request | Allow |
| Access token without DPoP proof | Reject |
| Bound token used with an unrelated key | Reject |
| Replayed DPoP proof | Reject |
| Incorrect `htm` | Reject |
| Incorrect `htu` | Reject |
| Insufficient scope | Reject |
| Expired access token | Reject |
| Revoked but otherwise unexpired access token | Reject |
| Token issued for another resource | Reject |

These scenarios are implemented in the security test suite and are
documented in `docs/security-test-matrix.md`.

## 7. Role in the Research

Baseline v1 serves as the comparison point for subsequent research.

~~~text
Baseline v1
    |
    | identify remaining delegation-related limitations
    v
Delegation-aware extension
    |
    | evaluate the same and additional attack scenarios
    v
Baseline vs. Proposed Approach
~~~

The proposed extension should add delegation-specific enforcement
without weakening the security guarantees already provided by
Baseline v1.

## 8. Reproducibility and Freeze Policy

Once Baseline v1 is merged and tagged, delegation-related features
must not be added to the baseline implementation.

Changes after the freeze should be limited to:

- confirmed bug fixes
- reproducibility fixes
- documentation corrections
- build or CI maintenance that does not change the baseline security model

Any such change should be explicitly documented if it affects
experimental comparison.

＝＝＝＝＝＝＝＝＝＝＝＝＝＝＝

# Baseline v1 定義

Baseline v1は、今後提案するDelegation-aware Enforcement方式と比較する
ための基準システムです。

Baselineでは、Tokenが正規であり、対象Resource向けであり、必要な権限を
持ち、有効かつ失効しておらず、TokenにBindingされた秘密鍵の保有者によって
使用されていることを検証します。

一方で、

- 誰が委譲したか
- 誰に委譲したか
- どのTaskのためか
- どのActionだけ許可されたか
- 再委譲が許可されるか

といったAgent Delegation固有の意味論はBaselineには含めません。

この境界を、Baseline v1と今後のProposed Extensionの比較基準とします。