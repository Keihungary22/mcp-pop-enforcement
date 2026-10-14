# Threat Model and Trust Boundaries

**Research:** Authorization-to-Execution Consistency in DPoP-Protected Delegated MCP Workflows
**Status:** Research hypothesis / feasibility investigation
**Related issue:** #26
**Research plan:** Plan 01 (Revised v2), October 10, 2026

## 1. Objective

This document defines the threat model for experiments E1-E4.

The investigated failure mode is a mismatch between an authorized invocation and the actual concrete resource modified by its committed execution.

The study specifically considers mutable resource aliases, changes in resource state after authorization, and time-of-check-to-time-of-use (TOCTOU) races.

It does not assume that OAuth, DPoP, or existing delegation techniques are themselves broken.

## 2. System and Participants

The planned experimental flow is:

```text
Human User
    |
    v
Agent A
    |
    | Attenuated delegation
    v
Agent B
    |
    | OAuth/DPoP-bound request
    v
Go Enforcement Gateway
    |
    | Access, DPoP, delegation, and invocation validation
    v
Trusted Execution Adapter
    |
    | Resolution / consistency control / commit
    v
Mock Calendar MCP Tool
    |
    v
Database
```

This is the conceptual experimental architecture. The exact placement of the adapter and MCP tool in the implementation must be documented without changing their trust responsibilities.

## 3. Assets and Security Objectives

Protected assets include:

- User-authorized calendar resources.
- Immutable Resource IDs.
- Mutable alias-to-resource mappings.
- Delegated operation permissions.
- Committed calendar event records.
- Authorization decision records.
- Authoritative execution and commit records.

Security objectives:

1. Preserve the immutable resource target fixed at authorization time.
2. Preserve operation-level authorization through execution.
3. Distinguish authorized invocations from actual committed effects.
4. Detect unauthorized writes reliably.
5. Preserve functionality for eligible legitimate invocations.

## 4. Trusted Components

For the initial experiments, the following components are assumed trusted within their specified responsibilities:

| Component | Trust assumption |
| --- | --- |
| Keycloak | Correctly issues and represents the configured OAuth authorization |
| Token and DPoP verification | Correctly validates the relevant token and proof properties |
| Go Enforcement Gateway | Correctly applies the configured validation logic of the selected baseline |
| Delegation verifier, when enabled | Correctly verifies chain validity and configured attenuation constraints |
| Trusted Execution Adapter | Executes operations according to its specified target-binding and concurrency behavior |
| Database atomicity primitives | Provide the documented transaction, conditional-update, or locking guarantees |
| MCP Tool contract | Correctly describes tool operations and their execution semantics |

Trust in the adapter does not imply that every experimental configuration is consistency-preserving. Weak configurations may intentionally re-resolve aliases or separate checks from writes to expose the hypothesized failure mode.

## 5. Trust Boundaries

### TB1 — Human Authorization to Agent Delegation

The user's authorized operation and resource must remain meaningful as authority passes to Agent A and Agent B.

A valid delegation chain is distinct from a correctly executed operation.

### TB2 — Agent Request to Go Enforcement Gateway

The request crosses from a potentially adversarial delegatee into the protected validation layer.

OAuth, DPoP, and delegation checks authenticate or authorize the request according to the selected baseline, but do not alone establish the final database modification.

### TB3 — Authorization Decision to Execution

This is the primary research boundary.

The Gateway permits a tool invocation, but later resource resolution and database execution may be influenced by mutable state.

The critical question is whether the authorized immutable target and allowed operation are preserved through the actual commit.

### TB4 — Alias Resolution and Concurrent State

Alias mappings may change between authorization, execution preparation, checks, and commit.

A non-atomic check followed by re-resolution can produce a different effective target.

### TB5 — Execution to Evidence

A Gateway `PERMIT` or HTTP success response is not authoritative evidence that the intended database write committed.

Actual outcomes must be established using commit records and backend state.

## 6. Attacker Model

An adversarial delegatee may:

- Hold a legitimate OAuth access token and its corresponding DPoP private key.
- Submit syntactically valid and cryptographically authorized MCP invocations.
- Select permitted request parameters within the applicable policy.
- Exploit a mutable resource alias where the baseline allows it.
- Coordinate with, or benefit from, another actor capable of changing an alias mapping.
- Trigger or exploit concurrency between authorization, resolution, checking, and commit.

Alias changes may also occur through legitimate concurrent activity without a malicious actor.

Possession of a valid token and DPoP key is an assumption in these scenarios, not evidence that DPoP has been compromised.

The attacker is not assumed to control every component. Alias mutation and other capabilities must be explicitly configured per scenario.

## 7. Concurrency and Race Assumptions

The experimental environment permits controlled interleavings involving:

1. Authorization at `t_auth`.
2. Initial alias resolution and/or version check.
3. Concurrent alias-mapping update.
4. Final execution or re-resolution.
5. Database commit or abort.

Experiments must use deterministic synchronization barriers or test hooks to position the state change at the intended point.

The research distinguishes:

- A check-then-use flow that re-resolves a mutable alias.
- A flow that directly uses the immutable Resource ID.
- A conditional write tied to an appropriate resource version.
- Atomic execution using transactions or locks.

These alternatives must not be assumed equivalent without evaluating their actual behavior and concurrency guarantees.

## 8. Representative Threat Scenarios

### E1 — Alias Swap

Initial alias mapping:

```text
work -> calendar_A
```

Authorization fixes `calendar_A` as the target.

The alias mapping changes:

```text
work -> calendar_B
```

A weak execution path re-resolves `work` and may commit to `calendar_B`.

Observe the authorization decision, actual commit target, execution outcome, and unauthorized commit count.

### E2 — Immutable Resource Binding

Use the E1 setup, but bind the authorized invocation to immutable `calendar_A`.

Evaluate whether an appropriately enforced existing invocation constraint prevents an unauthorized write to `calendar_B`.

Do not assume that argument validation alone is sufficient if execution subsequently resolves a different target.

### E3a — Re-resolve Race

The adapter checks an alias mapping and version.

A concurrent actor changes the mapping after the check but before the write.

The tool re-resolves the alias, potentially selecting a different resource.

Measure the outcome of this controlled TOCTOU interleaving.

### E3b — Immutable-ID Control

Keep the same initial state, alias change, and concurrency timing as E3a.

Change only the execution path so that the adapter writes directly to the immutable Resource ID fixed by authorization.

Use this comparison to isolate the significance of resource re-resolution.

### E4 — Atomic Execution

Separately evaluate:

- Direct immutable-ID execution.
- Version-based conditional update.
- Compare-and-swap.
- Database transactions.
- Lock-based execution.

For each approach, measure unauthorized commits, aborts, legitimate operation success, authorization latency, end-to-end latency, and concurrent behavior.

The presence of a transaction or lock alone is not sufficient evidence of correctness: the protected resource and the scope of atomicity must be specified.

## 9. Baseline Interpretation

- **B0:** OAuth + DPoP; no fine-grained delegation or invocation constraints.
- **B1:** Adds human-rooted delegation-chain validation.
- **B2-weak:** Constrains mutable aliases; used as a negative control.
- **B2-strong:** Uses correctly enforced invocation constraints, including immutable IDs when appropriate.
- **S:** Applies suitable existing execution-level consistency mechanisms.

A new proposed method is outside the initial implementation scope.

Research novelty cannot be established by showing only that a deliberately weak alias-based configuration fails.

## 10. In-Scope Behavior

- Correctly authenticated and DPoP-bound but potentially adversarial invocations.
- Valid and attenuated delegation chains where required by the baseline.
- Mutable alias-to-resource mappings.
- Resource identity preservation between authorization and commit.
- Authorization-to-execution target and operation consistency.
- Deterministic concurrent mapping changes and TOCTOU.
- Existing immutable-ID, versioning, transaction, CAS, and lock mitigations.
- Authoritative observation of writes, aborts, and failures.

## 11. Out-of-Scope Behavior

- Theft or compromise of signing keys.
- Forged Keycloak state or compromised token verification.
- Malicious backend reporting fabricated commit outcomes.
- Writes that bypass the assumed Trusted Execution Adapter.
- Natural-language task-to-policy generation.
- Distributed revocation.
- General business idempotency.
- A fully untrusted backend for which the Gateway must independently guarantee final resource writes.
- A new token format or authorization protocol in the initial phase.

This scope does not imply that excluded attacks are unimportant; they are outside the selected experimental model.

## 12. Observation and Evidence

Authorization and execution observations must be recorded separately.

Required values:

```text
invocation_id
scenario
baseline
authorization_decision: PERMIT | DENY
execution_outcome: COMMITTED | ABORTED | FAILED | TIMEOUT
authorized_resource_id
actual_commit_target
requested_operation
actual_operation
resource_version
commit_id
authorization_timestamp
execution_timestamp
commit_timestamp
```

Only an actual committed modification counts as a commit.

A timeout or failed response must not automatically be interpreted as proof of no committed write.

Resource IDs and operation permissions must be checked independently.

## 13. Assumptions and Limitations

- The initial environment uses deterministic mock agents, not live LLM-based decisions.
- Resource IDs remain stable and are not reused during experimental invocations.
- All relevant writes pass through the Trusted Execution Adapter.
- The database provides trustworthy state and commit observations.
- The model does not protect against an attacker with administrative control over the trusted runtime or storage.
- Findings must be evaluated against properly implemented existing mitigation mechanisms.
- The novelty of the research is not established in advance.
- The exact attacker capabilities and synchronization points must be reported for each experiment.
