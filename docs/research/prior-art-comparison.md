# Prior Art Comparison — Authorization-to-Execution Consistency

## Scope

This assessment compares the observed E1-E4 behavior against existing
authorization, delegation, and execution-consistency mechanisms.

The tested fixed-target property is:

    COMMITTED(i) => ActualTarget(i) == AuthorizedTarget(i)

This property concerns the final side effect, rather than merely
the validity of a credential or a prior authorization decision.

The experiments use a controlled SQLite research model with an
unsigned delegation fixture. They do not constitute a direct
security evaluation of the systems described in the cited work.

## 1. OAuth 2.0 DPoP — RFC 9449

Source:
https://www.rfc-editor.org/rfc/rfc9449.html

DPoP sender-constrains OAuth tokens by requiring possession of
the corresponding private key. The proof incorporates information
such as the HTTP method and URI, and protected resource requests
bind the proof to the access token.

Section 11.7 explicitly explains that DPoP does not provide
integrity protection for general request bodies and headers.

Relationship to E1-E4:

- DPoP provides sender-constrained credential presentation.
- It does not itself specify the immutable calendar resource that
  a downstream MCP Tool must ultimately modify.
- The B0 Gateway remains relevant authentication infrastructure.
- Passing DPoP validation does not establish that the final
  committed resource matches an authorized resource.

This is a distinction in security responsibilities, not a newly
discovered vulnerability in RFC 9449.

## 2. OAuth 2.0 Token Exchange — RFC 8693

Source:
https://www.rfc-editor.org/rfc/rfc8693.html

RFC 8693 defines token exchange and describes both delegation
and impersonation semantics. Delegation can retain information
about the subject and the actor exercising delegated authority.

Relationship to E1-E4:

- Token exchange can represent relevant delegation relationships.
- Actual authorization remains dependent on issued credentials,
  applicable policies, and resource-server enforcement.
- The RFC does not independently establish an invariant between
  a calendar resource authorized at one instant and a later
  side-effect target selected through a mutable alias.

No claim is made that compliant token-exchange deployments must
exhibit the tested mismatch.

## 3. Attenuating Authorization Tokens (AAT)

Source:
https://datatracker.ietf.org/doc/draft-niyikiza-oauth-attenuating-agent-tokens/

Reference version: draft-niyikiza-oauth-attenuating-agent-tokens-01,
June 2026. Internet-Draft; work in progress.

AAT defines signed, task-scoped delegation credentials.
The described capabilities include tool permissions,
argument constraints, delegation-chain verification, and
attenuation of authority across delegation steps.

Relationship to E1-E4:

- AAT is a stronger delegation model than the unsigned B1 fixture.
- Argument constraints are directly relevant to the question
  of which tool operation and target are authorized.
- This research has not implemented or attacked AAT verification.
- No conclusion about AAT's resistance to alias substitution
  can be inferred from the B2-strong + X1 experiment alone.

A credible comparison would need to specify how a verified
argument constraint is bound to the final execution target.

## 4. Verifiable Attenuated Delegation — Asor

Source:
https://datatracker.ietf.org/doc/draft-asor-wimse-agent-delegation-chain/

Reference version: draft-asor-wimse-agent-delegation-chain-01,
September 2026. Internet-Draft; work in progress.

This proposal specifies verifiable, attenuated delegation chains
for AI agents, using signed tokens, linked delegation evidence,
and offline verification of authority attenuation.

Relationship to E1-E4:

- The proposal addresses authority preservation across agents.
- It is a relevant strong existing delegation design.
- The experiments do not validate or attack its token format
  or delegation-chain verification procedure.
- Correct chain verification and correct final side-effect
  enforcement must not be conflated.

The observed failure of a deliberately re-resolving execution
mode is not evidence of a flaw in this proposal.

## 5. PAuth — Precise Task-Scoped Authorization for Agents

Source:
https://arxiv.org/abs/2603.17170

Sharma, Jiang, Lin, and Chen, 2026.

PAuth investigates precise task-scoped authorization for
agent actions. It introduces symbolic specifications of expected
service calls (NL slices) and provenance-linked operand values
(envelopes).

Relationship to E1-E4:

- PAuth is substantially closer to precise operation and
  argument authorization than conventional broad OAuth scopes.
- Its operand-provenance approach is relevant when authorization
  depends on how concrete arguments are produced.
- The present experiments do not implement PAuth envelopes
  or test a PAuth-compliant enforcement point.
- It is therefore unjustified to assert that E1-E4 identify
  an unresolved security gap in PAuth.

Any independent contribution involving operation arguments or
their transformations requires a more detailed comparison.

## 6. Existing execution-consistency controls

### Immutable resource-ID execution

Retaining the authorized immutable identifier and using that
identifier for the final write prevents alias-based target
substitution in the tested fixed-target model.

E2, E3, and E4 confirm this behavior in the research fixture.

This is an existing mitigation, not a new mechanism.

### Conditional SQL writes

A conditional write can enforce a relevant predicate at the
point of mutation. The E4 mapping-CAS fixture checks whether
the current alias still maps to the authorized identifier.

The tested implementation compares values rather than
monotonically increasing mapping versions.

Consequently, its behavior does not establish protection
against an A -> B -> A history.

### Checked transactions and locks

The E4 checked-transaction fixture verifies the mapping
and performs the write within a shared SQLite synchronization
and transaction boundary.

Its guarantee is limited to the specified predicate,
locking strategy, storage engine, and tested schedule.

### Transactions without authorization checks

A database transaction can make a set of writes atomic
without ensuring that the selected resource was authorized.

E4 demonstrates this distinction: the transaction-only
configuration commits to calendar_B after alias substitution.

This is a known implementation-level correctness issue,
not a discovery that database transactions are insecure.

## 7. Research findings versus existing techniques

| Finding | Interpretation | Novelty assessment |
| --- | --- | --- |
| E1: mutable alias substitution | Weak negative control | Known class of failure |
| E2: X1 vs X2 | Binding the verified target matters | Existing mitigation demonstrated |
| E3: check-then-re-resolve | TOCTOU in the tested execution flow | Known class of failure |
| E4: immutable-ID execution | Preserves fixed authorized target | Existing mitigation |
| E4: conditional mapping check | Detects changed mapping at mutation | Existing mitigation |
| E4: checked transaction | Protects a specified predicate within a synchronization boundary | Existing mitigation |
| E4: transaction-only mismatch | Atomicity alone does not imply correct authorization | Existing correctness distinction |

The B2-strong + X1 configuration checks an immutable ID but
does not preserve that ID for execution. It is not a counterexample
to correctly implemented immutable-ID binding.

## 8. Remaining research questions

The completed experiments have not demonstrated a new vulnerability
against strong existing authorization and execution controls.

A potentially independent research contribution would require
at least one of the following:

1. A clearly defined authorization invariant not already covered
   by existing task-scoped or argument-constrained enforcement.
2. A demonstrated failure under realistic conditions despite
   correctly applied existing controls.
3. A independently valuable formal correctness or evaluation
   framework that explains previously uncharacterized execution
   boundaries, supported by comparison with prior work.

These are candidate directions, not established contributions.

## 9. Provisional assessment

The present evidence supports REFINE as an intermediate research
checkpoint, not an unconditional decision to continue development.

If further comparison finds no independently valuable question,
NO-GO is the appropriate final conclusion.

A successful attack against deliberately weak alias re-resolution
is insufficient evidence of research novelty.

## References

1. RFC 9449, OAuth 2.0 Demonstrating Proof of Possession, 2023.
2. RFC 8693, OAuth 2.0 Token Exchange, 2020.
3. Niyikiza, Attenuating Authorization Tokens for Agentic
   Delegation Chains, Internet-Draft -01, 2026.
4. Asor, Verifiable Attenuated Delegation for AI Agent Chains,
   Internet-Draft -01, 2026.
5. Sharma et al., PAuth: Precise Task-Scoped Authorization
   for Agents, 2026.
