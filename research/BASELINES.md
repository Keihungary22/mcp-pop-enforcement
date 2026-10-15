# Issue #28 ? Authorization and Execution Baselines

## Research objective

Evaluate authorization-to-execution consistency in delegated,
DPoP-protected MCP workflows.

The core fixed-target safety condition is:

    COMMITTED(i) => ActualTarget(i) == AuthorizedTarget(i)

An execution must also use an authorized operation. A PERMIT decision
is not equivalent to a COMMITTED side effect.

The experiment distinguishes authorization decisions, execution
outcomes, and actual committed targets.

## Baseline definitions

| Baseline | Responsibility | Current implementation |
| --- | --- | --- |
| B0 | OAuth access-token and DPoP proof validation | Existing Go Gateway |
| B1 | Minimal Human -> Agent A -> Agent B delegation constraints | Fixed unsigned fixture |
| B2-weak | Resource constraint expressed using a mutable alias | Research authorization model |
| B2-strong | Resource constraint bound to an immutable calendar ID | Research authorization model |
| S | B2-strong authorization combined with X2 execution | Research runner |

B0 is not reimplemented in Python. The isolated research deployment
reuses the existing Go Gateway, with a separate Keycloak instance and
research-only MCP backend.

B1 does not implement a general-purpose, cryptographically verified
delegation protocol or Agent Authorization Token. The delegation chain
is a controlled experimental fixture, not a trusted authorization
credential.

## Execution models

- X1 (alias): resolve the mutable alias during execution.
- X2 (immutable): execute against the immutable authorized resource ID.
- X3 (atomic): reserved for the subsequent concurrency experiments.

Authorization and execution occur at separate logical points.

## Comparison matrix

| Authorization | Execution | Expected interpretation |
| --- | --- | --- |
| B2-weak | X1 | Mutable authorization reference and mutable execution lookup |
| B2-strong | X1 | Immutable-ID policy, but execution may still re-resolve the alias |
| B2-weak | X2 | Execution uses an immutable target despite the weak policy representation |
| B2-strong | X2 | Strong authorization constraint plus immutable execution target (S) |

These are configurations to be evaluated, not claims that every
configuration prevents unauthorized side effects.

## Fixed-target example

At authorization:

    work -> calendar_A
    AuthorizedTarget = calendar_A

After authorization, the mapping changes:

    work -> calendar_B

X1 may write to calendar_B if it re-resolves `work`.
X2 writes using the authorized immutable ID, calendar_A.

Under the fixed-target policy, a commit to calendar_B violates the
authorized-target consistency condition.

A dynamic-alias policy is a different policy and must not be silently
treated as fixed-target authorization.

## Trust boundaries and limitations

1. The Go Gateway validates OAuth/DPoP credentials before forwarding
   authenticated requests. B0 does not itself establish authorization
   for the final calendar side effect.
2. Research MCP's authorization runner evaluates a fixed delegation
   fixture and experimental resource constraints.
3. The research MCP tool currently accepts baseline, execution mode,
   and authorized_resource_id from the caller. These are experimental
   inputs, not server-authenticated delegation claims.
4. Removing the raw calendar tool from MCP registration prevents that
   particular published-tool bypass; it does not establish a complete
   production-grade authorization boundary.
5. The research server, SQLite calendar environment, and test fixtures
   are independent of the existing baseline implementation.
6. X3 atomicity, adversarial concurrency evaluation, and broader attack
   coverage are deferred to subsequent issues.

## Verification status

Completed locally:

- Research Keycloak startup and realm import.
- Go Gateway health check and unauthenticated HTTP 401 response.
- DPoP-bound token acquisition and authenticated MCP call through
  the existing Go Gateway to the isolated research backend.
- One successful B0 end-to-end event committed to calendar_A.
- In-process MCP checks for research authorization PERMIT, DENY,
  committed target, and no commit following DENY.
- Research and non-security regression tests.

Not established by the above tests:

- Full B0 rejection matrix against the isolated deployment.
- Cryptographic validation of the B1 delegation fixture.
- Server-trusted enforcement of caller-selected experiment settings.
- Production safety under concurrent alias updates.
- End-to-end execution of the updated MCP tool after the latest
  research container rebuild.

## Isolation

Research files are confined to `research/` and `tests/research/`.
The baseline Go Gateway, mock MCP, and root Docker Compose configuration
are not modified.

The research deployment is defined in `research/compose.yaml`.
