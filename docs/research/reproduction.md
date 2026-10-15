# Research Experiment Reproduction

## Environment

Run from the repository root using the project's Python environment.
E1-E4 use the isolated SQLite research model; Docker is not required.

## Commands

Run the research tests:

    python -m pytest tests/research -q

Regenerate E1/E2 results:

    python -m research.e1_e2

Regenerate E4 results:

    python -m research.e4

Run the four E3 schedules:

    python -c 'from research.e3 import run_e3; [print(run_e3(m, swap=s)) for m in ("alias", "immutable") for s in (False, True)]'

The E3 command prints results but does not regenerate e3.json.
The saved E3 artifact was produced by running the four schedules
and recording their outputs.

## Verification

- E1/E2 pause after authorization.
- E3 pauses after the resource check.
- E4 compares four existing execution/atomicity controls.
- SQLite events and commit logs provide committed-target evidence.
- Experiment counts are kept separate because configurations overlap.

## Limitations

The results cover deterministic research schedules only.
Do not include artificial barrier waiting time in latency results.
No production-level security or cryptographic delegation guarantee
follows from these tests.
