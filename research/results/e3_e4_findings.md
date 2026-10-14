# Issue #30 ? E3/E4 Findings

## Scope and experimental schedule

Fixed authorized target: `calendar_A`.
Initial mapping: `work -> calendar_A`.

E3 uses explicit synchronization after the preliminary mapping
check. In swap cases, the test changes `work -> calendar_B`
before the worker resumes. No arbitrary sleep determines the race.

E4 follows authorization, resource check, optional alias change,
execution, and commit or abort. Its synchronization is controlled
by a test barrier.

## E3 observed results

| Experiment | Scenario | Mode | Decision | Outcome | Target | Unauthorized commits |
| --- | --- | --- | --- | --- | --- | ---: |
| E3a | control | X1 | PERMIT | COMMITTED | calendar_A | 0 |
| E3a | swap | X1 | PERMIT | COMMITTED | calendar_B | 1 |
| E3b | control | X2 | PERMIT | COMMITTED | calendar_A | 0 |
| E3b | swap | X2 | PERMIT | COMMITTED | calendar_A | 0 |

E3a re-resolves the mutable alias after an earlier mapping check.
The swapped run commits to calendar_B. E3b retains the authorized
immutable ID and commits to calendar_A under the same schedule.

## E4 observed results

| Control | Scenario | Outcome | Target | Unauthorized commits | Aborts |
| --- | --- | --- | --- | ---: | ---: |
| immutable | control | COMMITTED | calendar_A | 0 | 0 |
| immutable | swap | COMMITTED | calendar_A | 0 | 0 |
| mapping_cas | control | COMMITTED | calendar_A | 0 | 0 |
| mapping_cas | swap | ABORTED | None | 0 | 1 |
| checked_transaction | control | COMMITTED | calendar_A | 0 | 0 |
| checked_transaction | swap | ABORTED | None | 0 | 1 |
| transaction_only | control | COMMITTED | calendar_A | 0 | 0 |
| transaction_only | swap | COMMITTED | calendar_B | 1 | 0 |

## Security and functionality

- E3 Unauthorized Commit Count: 1.
- E3 legitimate controls: 2/2.
- E4 Unauthorized Commit Count: 1.
- E4 Target Mismatch Count: 1.
- E4 Aborted Operation Count: 2.
- E4 False Denial Count: 0.
- E4 Legitimate Operation Success Rate: 100% (4 tested controls).

E3 and E4 include overlapping configurations. Their counts are
separate observations, not independent vulnerability discoveries.

## Existing mitigation comparison

- Immutable-ID direct execution maintains the fixed authorized target.
- Mapping CAS uses a conditional SQL insert and aborts if the current
  alias mapping differs. It does not use a version counter and does
  not protect against an A -> B -> A ABA history.
- Checked Transaction checks the mapping and writes inside the same
  SQLite lock and transaction. It aborts on a detected mismatch.
- Transaction Only performs an atomic write but does not enforce the
  authorized target; it commits to calendar_B in the swapped case.

The mapping-sensitive controls enforce an additional condition:
the current alias mapping must still match the authorized ID.
That is distinct from the fixed-target property preserved by X2.

## Performance measurement

Authorization latency: NOT MEASURED.
Execution latency: NOT MEASURED.
End-to-end latency: NOT MEASURED.
p95 latency: NOT MEASURED.
Throughput: NOT MEASURED.

Deterministic barriers deliberately suspend execution, so their
waiting time cannot be treated as service latency. Any future
performance benchmark must run separately without artificial pauses
and report its workload, repetitions, timing boundaries, and setup.

## Research interpretation ? provisional REFINE

The observed mismatch occurs when an implementation discards the
previously checked resource ID and re-resolves a mutable alias.
Correctly maintained immutable-ID execution prevents this target
substitution in the tested model.

These experiments do not demonstrate a failure of strong existing
immutable-ID binding, nor do they establish research novelty.
Before proposing new enforcement, any further work must identify
an additional changing authorization condition and the specific
atomicity requirement needed to preserve it.

## Limitations

- The delegation fixture is unsigned and simulated.
- The experiments invoke the research model, not the live Gateway.
- SQLite is in-memory and shared through an in-process RLock.
- The schedules are controlled and do not cover arbitrary races.
- CAS is value-based, not version-based.
- No distributed transaction or production lock guarantee is tested.
- No performance numbers have been collected.
