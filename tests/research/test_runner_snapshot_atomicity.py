import threading
from unittest.mock import patch

import research.runner as runner_module
from research.calendar_lab.lab import CalendarLab
from research.runner import ResearchRunner


def test_authorization_snapshot_is_atomic_with_policy_check():
    lab = CalendarLab()
    runner = ResearchRunner(lab)

    real_authorize = runner_module.authorize
    attempted = threading.Event()
    completed = threading.Event()
    workers = []

    def tracked_authorize(**kwargs):
        def remap():
            attempted.set()
            lab.set_alias("work", "calendar_B")
            completed.set()

        worker = threading.Thread(target=remap, daemon=True)
        workers.append(worker)
        worker.start()

        assert attempted.wait(timeout=2)
        assert not completed.wait(timeout=0.1)

        return real_authorize(**kwargs)

    with patch.object(runner_module, "authorize", side_effect=tracked_authorize):
        result = runner.run(
            invocation_id="atomic-auth-snapshot",
            baseline="B2-strong",
            mode="immutable",
        )

    for worker in workers:
        worker.join(timeout=2)
        assert not worker.is_alive()

    assert completed.is_set()
    assert result["authorization_decision"] == "PERMIT"
    assert result["actual_commit_target"] == "calendar_A"
    assert runner.authorization_records[0]["alias_at_authorization"] == "calendar_A"
    assert lab.resolve("work") == "calendar_B"
