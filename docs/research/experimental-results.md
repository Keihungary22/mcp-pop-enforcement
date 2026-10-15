# Experimental Results ? Issues #29?#31

## Research hypothesis

Fixed-target safety condition:

`COMMITTED(i) => ActualTarget(i) == AuthorizedTarget(i)`

Initial alias: `work -> calendar_A`.
Authorized target: `calendar_A`.
Swapped alias: `work -> calendar_B`.

The experiments evaluate whether mutable alias resolution
can create a mismatch between authorization and actual side effects.

The delegation chain is an unsigned research fixture.
These experiments use an isolated in-memory SQLite environment.

## E1/E2 ? Alias Swap and Immutable Resource Binding

Hypothesis: alias re-resolution after authorization may change
the committed target; immutable-ID execution should preserve it.

Schedule: authorization -> pause -> optional alias change
-> execution -> database commit inspection.

| Experiment | Case | Baseline | Mode | Scenario | Authorization | Outcome | Commit target | Unauthorized |
| --- | --- | --- | --- | --- | --- | --- | --- | ---: |
| E1 | E1 | B2-weak | X1 | alias_swap | PERMIT | COMMITTED | calendar_B | 1 |
| E2 | A | B2-weak | X1 | alias_swap | PERMIT | COMMITTED | calendar_B | 1 |
| E2 | A | B2-weak | X1 | legitimate_control | PERMIT | COMMITTED | calendar_A | 0 |
| E2 | B | B2-strong | X1 | alias_swap | PERMIT | COMMITTED | calendar_B | 1 |
| E2 | B | B2-strong | X1 | legitimate_control | PERMIT | COMMITTED | calendar_A | 0 |
| E2 | C | B2-weak | X2 | alias_swap | PERMIT | COMMITTED | calendar_A | 0 |
| E2 | C | B2-weak | X2 | legitimate_control | PERMIT | COMMITTED | calendar_A | 0 |
| E2 | D | B2-strong | X2 | alias_swap | PERMIT | COMMITTED | calendar_A | 0 |
| E2 | D | B2-strong | X2 | legitimate_control | PERMIT | COMMITTED | calendar_A | 0 |

- E1 unauthorized commits: 1.
- E2 unauthorized commits: 2.
- E2 legitimate controls: 4/4.

E1 and E2-A repeat the same experimental configuration.
Their counts are not independent vulnerability discoveries.

## E3 ? TOCTOU

Hypothesis: alias re-resolution after an earlier resource check
can select a different final target.

Schedule: authorization -> resource check -> pause
-> optional alias change -> execution -> commit inspection.

| Experiment | Scenario | Mode | Authorization | Outcome | Commit target | Unauthorized |
| --- | --- | --- | --- | --- | --- | ---: |
| E3a | control | X1 | PERMIT | COMMITTED | calendar_A | 0 |
| E3a | swap | X1 | PERMIT | COMMITTED | calendar_B | 1 |
| E3b | control | X2 | PERMIT | COMMITTED | calendar_A | 0 |
| E3b | swap | X2 | PERMIT | COMMITTED | calendar_A | 0 |

- E3 unauthorized commits: 1.
- E3 legitimate controls: 2/2.

## E4 ? Existing Atomicity Controls

Hypothesis: direct immutable-ID execution, conditional writes,
checked transactions, and transaction-only execution provide
different correctness guarantees.

Schedule: authorization -> resource check -> pause
-> optional alias change -> execution -> commit or abort.

| Control | Scenario | Authorization | Outcome | Commit target | Unauthorized | Abort | False denial |
| --- | --- | --- | --- | --- | ---: | ---: | ---: |
| immutable | control | PERMIT | COMMITTED | calendar_A | 0 | 0 | 0 |
| immutable | swap | PERMIT | COMMITTED | calendar_A | 0 | 0 | 0 |
| mapping_cas | control | PERMIT | COMMITTED | calendar_A | 0 | 0 | 0 |
| mapping_cas | swap | PERMIT | ABORTED | None | 0 | 1 | 0 |
| checked_transaction | control | PERMIT | COMMITTED | calendar_A | 0 | 0 | 0 |
| checked_transaction | swap | PERMIT | ABORTED | None | 0 | 1 | 0 |
| transaction_only | control | PERMIT | COMMITTED | calendar_A | 0 | 0 | 0 |
| transaction_only | swap | PERMIT | COMMITTED | calendar_B | 1 | 0 | 0 |

- E4 unauthorized commits: 1.
- E4 target mismatches: 1.
- E4 aborted operations: 2.
- E4 false denials: 0.
- E4 execution failures in tested cases: 0.
- E4 legitimate operation success rate: 100% (4/4 controls).

## Security interpretation

- B2-strong + X1 checks an immutable ID during authorization
  but re-resolves the alias to select the execution target.
- X2 retains the authorized ID and prevents target substitution
  under the tested schedules.
- Mapping CAS and checked transactions reject changed mappings.
- Transaction-only execution does not itself preserve the
  authorized target.

B2-strong + X1 is not a counterexample to a correctly implemented
immutable-ID binding maintained through execution.

E1-E4 include overlapping conditions, so unauthorized commit
counts must not be added as independent vulnerability discoveries.

## Performance

Authorization latency: NOT MEASURED.
End-to-end latency: NOT MEASURED.
p95 latency: NOT MEASURED.
Throughput: NOT MEASURED.
Mitigation overhead: NOT MEASURED.

Artificial synchronization delays are excluded from performance
claims. No separate performance benchmark was conducted.

## Limitations

- Deterministic schedules do not prove arbitrary race safety.
- SQLite is in-memory and process-local.
- Delegation is simulated and unsigned.
- Mapping CAS is value-based rather than version-based.
- ABA histories and distributed atomicity were not tested.
- No new vulnerability or security mechanism is established.

## Source artifacts

- `research/results/e1_e2.json`
- `research/results/e3.json`
- `research/results/e4.json`
- `research/results/e1_e2_findings.md`
- `research/results/e3_e4_findings.md`
