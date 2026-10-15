# Research Decision — Issue #31

## Decision: REFINE — Reposition

The E1-E4 experiments demonstrate authorization-to-execution target
mismatches under deliberately vulnerable alias re-resolution
configurations.

However, the experiments do not establish a novel vulnerability
against correctly implemented strong enforcement mechanisms.

The research is therefore repositioned toward defining and evaluating
authorization-to-execution consistency conditions in delegated
MCP workflows.

REFINE is a research-direction decision, not a claim that an
independently novel contribution has already been demonstrated.

## 1. Evidence from Completed Experiments

### E1

The initial alias-substitution experiment produced one unauthorized
commit to calendar_B instead of the authorized calendar_A.

This reproduces a known mutable-reference failure mode.

### E2

The four authorization/execution combinations were evaluated under
normal and alias-swap conditions.

- B2-weak + X1: unauthorized commit after alias substitution.
- B2-strong + X1: unauthorized commit after alias substitution.
- B2-weak + X2: authorized target preserved.
- B2-strong + X2: authorized target preserved.

The strong authorization check alone was insufficient when X1
discarded the checked target and independently re-resolved the alias.

This does not demonstrate a failure of correctly maintained
immutable-ID binding.

Recorded unauthorized commits: 2.
Legitimate control operations: 4/4 successful.

### E3

Deterministic scheduling between mapping verification and execution
reproduced a target mismatch under X1.

X2 retained the authorized target under the tested schedules.

Recorded unauthorized commits: 1.
Legitimate control operations: 2/2 successful.

### E4

Four existing enforcement approaches were compared:

- Immutable ID: committed to authorized calendar_A.
- Mapping CAS: aborted when the alias mapping no longer matched.
- Checked transaction: aborted when the checked condition failed.
- Transaction only: committed to calendar_B after alias substitution.

Recorded unauthorized commits: 1.
Recorded target mismatches: 1.
Recorded aborts: 2.
Recorded unchanged-mapping aborts: 0.
Legitimate no-change operations: 4/4 successful.

The unchanged-mapping abort count measures aborts in scenarios
without an alias change. It is not a policy-independent
false-denial metric.

All four experimental stages were completed. No E1-E4 experiment
is classified as skipped.

## 2. Security Interpretation

The current evidence supports the distinction between:

1. Correct authorization decisions.
2. Preservation of the authorized target through execution.
3. Optional mapping-consistency requirements at commit time.

Existing immutable-ID execution prevents the tested fixed-target
substitution.

Conditional writes and checked transactions can additionally
enforce mapping-sensitive conditions.

Transaction atomicity alone does not establish resource authorization.

The tested Mapping CAS is value-based and does not establish
history-sensitive protection against ABA changes.

## 3. Functionality Interpretation

Legitimate operations succeeded in all recorded no-change controls:

- E2: 4/4
- E3: 2/2
- E4: 4/4

These are separate experimental observations with overlapping
configurations, not ten independent production workloads.

Future comparisons must distinguish:

- Authorization denial
- Concurrency or policy-condition abort
- Execution failure
- Successful, policy-consistent commit

An abort is not automatically a false denial.
Classification depends on the common authorization semantics.

## 4. Performance

NOT MEASURED.

Authorization latency, end-to-end latency, p95 latency,
throughput, and mitigation overhead were not collected
in a separate meaningful benchmark.

The experiments prioritized deterministic security schedules
and evaluation of the research hypothesis.

Artificial synchronization waiting time must not be used
as a performance measurement.

No performance superiority claim is made.

## 5. Prior Art and Novelty

Relevant prior work includes RFC 9449, RFC 8693,
AAT, Asor, PAuth, and established TOCTOU mitigations.

The completed experiments show behavior consistent with known
execution-consistency and concurrency principles.

No new authorization protocol, bypass of correctly implemented
strong controls, or independently novel security guarantee
has been established.

See prior-art-comparison.md for the source comparison.

## 6. Repositioned Research Question

Under which conditions are delegated authorization constraints
preserved from the delegation grant and MCP invocation through
tool-side resolution to the actual committed side effect?

The proposed research separates:

P1 — Execution Target Consistency:

    COMMITTED(i) => ActualTarget(i) == AuthorizedTarget(i)

P2 — Operation Consistency:

    COMMITTED(i) => ActualOperation(i) in AuthorizedOperations(i)

P3 — Mapping Endpoint Consistency:

    COMMITTED(i) => MappingAtCommit(i) == MappingAtAuthorization(i)

P1 was tested in E1-E4 under the fixed-target model.

P2 has not been independently evaluated.

P3 compares endpoint mappings; it does not exclude an
intermediate A -> B -> A mapping history.

The research must demonstrate what its cross-component model
and evaluation methodology add beyond ordinary TOCTOU testing
and existing delegated-authorization approaches.

## 7. Next Research Checkpoint

The next stage is limited to:

1. Constructing a minimal state-transition model for P1.
2. Mapping existing E1-E4 traces onto that model.
3. Identifying sufficient assumptions for P1 preservation.
4. Comparing the proposed model and evaluation capability
   against prior art.
5. Reassessing whether a defensible independent contribution
   is plausible.

A second MCP tool and performance benchmarking are conditional
on that checkpoint.

This issue does not claim that those later activities are complete.

## 8. Limitations

- Experimental delegation is an unsigned research fixture.
- SQLite execution is isolated and process-local.
- Tested schedules are deterministic.
- Arbitrary concurrency and distributed execution are untested.
- AAT, Asor, and PAuth implementations were not attacked.
- No production security guarantee is established.
- Novelty of the proposed repositioned research is unresolved.

## 9. Reproduction and Preservation

Experimental implementations, automated tests, negative results,
and JSON artifacts must be preserved.

Supporting documents:

- experimental-results.md
- prior-art-comparison.md
- reproduction.md

Experimental artifacts:

- research/results/e1_e2.json
- research/results/e3.json
- research/results/e4.json

Research implementations must remain isolated from the stable
OAuth/DPoP Gateway.

## Conclusion

Decision: REFINE.

The initial alias-substitution hypothesis is insufficient as
a standalone novelty claim.

The investigation continues only as a repositioned study of
delegated authorization-to-execution consistency conditions
and their reproducible evaluation.

The next novelty checkpoint may support continued research,
further refinement, or termination.

This issue records the conclusion of the original E1-E4
experimental phase, not approval of a finished research paper.
