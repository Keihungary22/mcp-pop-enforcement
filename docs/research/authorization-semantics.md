# Authorization Semantics and Evaluation Properties

**Research:** Authorization-to-Execution Consistency in DPoP-Protected Delegated MCP Workflows
**Status:** Research hypothesis / feasibility investigation
**Related issue:** #26
**Research plan:** Plan 01 (Revised v2), October 10, 2026

## 1. Purpose and Scope

This document defines the authorization semantics and measurable properties used throughout experiments E1-E4.

The research investigates whether a resource authorized at invocation time remains the actual target of a committed MCP tool operation.

The study does not propose a new token format or authorization protocol at this stage. Its objective is to identify the conditions under which existing authorization constraints and execution mechanisms preserve authorization-to-execution consistency.

## 2. Authorization Layers

The following mechanisms address distinct security questions:

| Layer | Responsibility | Limitation |
| --- | --- | --- |
| OAuth 2.0 | Determines access-token authorization, including audience and scope | Does not directly establish the actual resource modified by a tool |
| DPoP | Demonstrates possession of the key associated with a token and binds proofs to HTTP requests | Does not directly bind the final database side effect |
| Delegation Chain | Validates human-rooted delegation, attenuation, and delegated authority | Does not by itself guarantee the resource ultimately modified |
| Invocation Constraints | Restricts permitted tools, operations, arguments, and resource identifiers | Effectiveness depends on correct enforcement and execution semantics |
| Trusted Execution Adapter | Applies authorized operations to concrete resources | Must preserve the authorized resource and operation through commit |

The experimental comparison distinguishes access authorization, proof-of-possession, delegation validity, invocation authorization, and execution consistency.

## 3. Roles

- **Human User:** Originates a delegation granting specific authority over resources and operations.
- **Agent A:** Receives the user's delegation and may delegate attenuated authority.
- **Agent B:** Acts as the delegatee and submits MCP tool invocations.
- **Go Enforcement Gateway:** Validates applicable OAuth, DPoP, delegation, and invocation constraints according to the experimental baseline.
- **Trusted Execution Adapter:** Resolves resource references and executes operations under the selected consistency mechanism.
- **MCP Tool:** Exposes calendar operations and invokes the trusted execution path.
- **Database:** Stores resources, aliases, versions, events, and committed effects.

These roles define the planned experimental architecture, not a claim that every component has already been implemented.

## 4. Resource Identity and Alias Semantics

A **Resource ID** is a stable identifier of a concrete calendar resource, such as `calendar_A`.

An **alias** is a mutable name, such as `work`, whose mapping may change over time.

Let:

- `Resolve(alias, t)` denote the resource identified by an alias at time `t`.
- `t_auth` denote the authorization time.
- `t_exec` denote the execution time.

For example:

```text
At authorization: Resolve("work", t_auth) = calendar_A
At execution:     Resolve("work", t_exec) = calendar_B
```

### 4.1 Fixed-Target Authorization

The initial experiments use fixed-target semantics.

The user authorizes an operation on the concrete immutable Resource ID selected at authorization time.

```text
AuthorizedTarget(i) = Resolve("work", t_auth) = calendar_A
```

A committed write to `calendar_B` is unauthorized, even if the alias `work` refers to `calendar_B` at execution time.

### 4.2 Dynamic-Alias Authorization

Dynamic-alias authorization is a distinct policy:

```text
AuthorizedTarget(i) = Resolve("work", t_exec)
```

Under that intentionally selected policy, a write to `calendar_B` may be authorized.

The initial experiments do not evaluate this interpretation as a violation of fixed-target semantics. The two policies must not be conflated.

## 5. Invocation Authorization

An invocation is identified by `i`.

For each invocation, record:

- The authorizing principal and delegation context.
- The requested tool and operation.
- The requested alias, when applicable.
- The authorized immutable Resource ID.
- The permitted operation and its relevant constraints.
- The actual resource affected by execution.

An authorization decision is either `PERMIT` or `DENY`.

`PERMIT` does not imply that a database write occurred.

A denied invocation must not generate a committed side effect through the protected execution path.

## 6. Security Properties

### 6.1 Authorization-to-Execution Target Consistency

The primary security property is:

```text
Committed(i) => ActualTarget(i) == AuthorizedTarget(i)
```

Definitions:

- `Committed(i)`: Invocation `i` produced an actual committed write.
- `AuthorizedTarget(i)`: The immutable Resource ID fixed at authorization time.
- `ActualTarget(i)`: The immutable Resource ID actually modified by the committed write.

An **Unauthorized Target Commit** is a committed write where:

```text
ActualTarget(i) != AuthorizedTarget(i)
```

A rejected, failed, timed-out, or aborted invocation is not counted as an unauthorized commit unless an actual unauthorized write was committed. A timeout alone does not prove that no commit occurred.

### 6.2 Operation Authorization Consistency

Target equality alone is insufficient.

For every committed invocation, the committed operation must also be permitted under the applicable authorization constraints:

```text
Committed(i) => OperationAllowed(i, ActualOperation(i), ActualTarget(i))
```

For example, permission to read `calendar_A` does not authorize writing to `calendar_A`.

The target-consistency property and operation-authorization property are evaluated separately.

### 6.3 Delegation and Access Preconditions

Where the baseline requires them, valid OAuth authorization, DPoP verification, and delegation constraints are preconditions for permitting an invocation.

These checks are not treated as proof of the final execution target.

## 7. Execution Outcomes

Record authorization and execution independently.

Authorization decisions:

- `PERMIT`
- `DENY`

Execution outcomes:

- `COMMITTED`
- `ABORTED`
- `FAILED`
- `TIMEOUT`

A committed outcome requires evidence of a committed effect, not merely a successful HTTP response.

For ambiguous failures or timeouts, consult authoritative commit records before classifying whether a write occurred.

## 8. Evaluation Metrics

### 8.1 Security Metrics

**Unauthorized Commit Count (UCC)**

Number of invocations with a committed write that violates the fixed-target or operation-authorization constraints. Target violations and operation violations are also reported separately.

**Unauthorized Commit Rate (UCR)**

```text
UCR = Unauthorized Commit Count / Evaluated Write Invocations
```

The denominator is the number of evaluated write invocations in the declared experimental population. Both the numerator and denominator must be reported.

**Target Mismatch Count**

Number of committed invocations for which:

```text
ActualTarget(i) != AuthorizedTarget(i)
```

Abort and timeout outcomes must be reported separately rather than silently treated as safe successful operations.

### 8.2 Functionality Metrics

**Legitimate Operation Success Rate (LOSR)**

```text
LOSR = Successful Legitimate Invocations / Eligible Legitimate Invocations
```

A successful legitimate invocation must commit the intended authorized operation to the correct resource.

A configuration that rejects or aborts every request may produce zero unauthorized commits but has LOSR = 0 when eligible legitimate invocations exist.

### 8.3 Operational Metrics

Where relevant, measure:

- Authorization latency.
- End-to-end latency.
- Aborted operation count.
- Timeout and failure counts.
- Concurrent request behavior.

Use consistent workloads and measurement boundaries when comparing alternatives.

## 9. Required Observation Records

Each invocation must have a correlation identifier linking authorization and execution records.

Minimum fields:

| Field | Meaning |
| --- | --- |
| `invocation_id` | Correlation identifier |
| `scenario` | E1, E2, E3a, E3b, or E4 |
| `baseline` | B0, B1, B2-weak, B2-strong, or S |
| `requested_alias` | Mutable alias used in the request, if applicable |
| `authorized_resource_id` | Immutable ID fixed at authorization time |
| `requested_operation` | Tool operation submitted by the delegatee |
| `authorized_operation` | Operation allowed by the authorization policy |
| `authorization_decision` | PERMIT or DENY |
| `execution_outcome` | COMMITTED, ABORTED, FAILED, or TIMEOUT |
| `actual_commit_target` | Resource ID actually modified, if committed |
| `actual_operation` | Operation actually committed, if any |
| `resource_version` | Relevant version before and/or after execution |
| `commit_id` | Authoritative commit identifier, if committed |
| `timestamps` | Authorization, execution, and commit timestamps where available |

An authorization log is not a substitute for a commit log.

Commit records must be correlated with the actual backend state to identify committed side effects reliably.

## 10. Experimental Baselines

| Baseline | Definition | Research purpose |
| --- | --- | --- |
| B0 | OAuth + DPoP | Establish whether token and proof validation alone preserve execution consistency |
| B1 | B0 + human-rooted Delegation Chain | Separate delegation validity from execution consistency |
| B2-weak | B1 + constraints referencing mutable aliases | Negative control for alias-related inconsistency |
| B2-strong | Correct invocation constraints, including immutable Resource IDs where appropriate | Evaluate strong existing invocation-level authorization |
| S | Strong existing mitigations such as direct immutable-ID execution, conditional updates, CAS, transactions, and locks | Determine whether established execution mechanisms already provide the required guarantees |

B2-weak is a negative control, not the principal competing state-of-the-art solution.

No complete AAT specification compliance is claimed without separate conformance assessment.

A proposed method (P) is considered only if meaningful limitations of strong existing solutions are demonstrated.

## 11. Experimental Consistency Requirements

E1-E4 must use the same fixed-target authorization semantics.

- **E1:** Evaluate alias replacement after authorization.
- **E2:** Evaluate binding to immutable Resource IDs under the same alias change.
- **E3a:** Evaluate re-resolution after a non-atomic check.
- **E3b:** Use direct immutable-ID execution as a controlled comparison to E3a.
- **E4:** Evaluate atomic and consistency-preserving existing mechanisms separately.

Concurrency must be reproduced deterministically with barriers or test hooks rather than relying solely on arbitrary sleeps.

## 12. Assumptions and Limitations

- The initial study uses deterministic mock agents and calendar tools rather than LLM-generated decisions.
- Resource IDs are treated as stable and not reused during the experiment.
- The Trusted Execution Adapter and its relevant enforcement logic are assumed to behave according to their specified contracts.
- The backend provides authoritative evidence of committed writes.
- The initial work does not introduce a new authorization protocol.
- The threat model and its excluded attacker capabilities are defined separately in `threat-model.md`.
- Novelty is not established at this stage.
