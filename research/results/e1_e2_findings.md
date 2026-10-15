# Issue #29 — E1/E2 Initial Findings

## Experimental conditions

- Fixed authorized target: `calendar_A`.
- Initial mapping: `work -> calendar_A`.
- Swap scenarios: mapping changes to `calendar_B` after authorization.
- Control scenarios: mapping remains unchanged.
- Synchronization: `PausePoint`; no timing-based race assumptions.
- Evidence: SQLite event rows, commit logs and execution logs.

## Observed results

| Experiment | Case | Baseline | Mode | Scenario | Decision | Outcome | Actual target | Unauthorized commits |
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

## Metrics

- E1 Unauthorized Commits: **1**
- E2 Unauthorized Commits: **2**
- E1 and E2-A repeat the same configuration; results are not independent.
- Legitimate Operation Success Rate: **4/4 (100%)**

## Feasibility assessment

These measurements characterize the implemented research model only.
A mismatch in B2-weak + X1 is a negative-control result,
not evidence of research novelty.
The combined B2-strong + X2 configuration represents the
strong existing control for this comparison.

E3/E4 may be justified by a separately identified question about
execution boundaries or concurrency; this experiment alone
does not establish that such a question is novel.

## Preliminary research decision: REFINE

Decision: **Refine (provisional)**.

Both X1 configurations committed to calendar_B after authorization
for calendar_A. Both X2 configurations preserved calendar_A.
Authorization-time ID validation alone did not preserve the target
when execution independently re-resolved the mutable alias.

B2-strong + X1 does not invalidate correctly maintained immutable-ID
binding: this configuration discards the verified ID at execution.
These results demonstrate required consistency conditions,
not a novel vulnerability or security mechanism.

The legitimate controls succeeded in 4/4 tested cases (100%).
This does not establish safety under arbitrary concurrency.

Before E3/E4, define which additional authorization condition
can change and what atomicity is required to preserve it.
Reassess novelty against existing research.

## Limitations

- The delegation chain is an unsigned fixture.
- This suite directly invokes the research runner, not the Go Gateway.
- The calendar database is isolated and in-memory.
- Experiment settings are controlled by the test harness.
- No production-level security claim follows from these results.
- Continue / Refine / Early No-Go requires interpreting the
  observations against existing research and security controls.
