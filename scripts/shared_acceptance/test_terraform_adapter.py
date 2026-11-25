import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

import terraform_adapter as adapter


class VerifyFixtureTest(unittest.TestCase):
    def test_failed_command_reports_diagnostic(self):
        with self.assertRaisesRegex(RuntimeError, "fixture diagnostic"):
            adapter._run(["python3", "-c", "import sys; print('fixture diagnostic'); sys.exit(2)"], cwd=".")

    def test_ci_pins_match_runner(self):
        root = Path(__file__).resolve().parents[2]
        workflow = (root / ".github/workflows/shared-scenario-prototype.yml").read_text()
        self.assertIn("ref: " + adapter.CONTRACT_REV, workflow)
        self.assertIn("git -C provider archive " + adapter.SOURCE_REV, workflow)

    def test_rejects_wrong_revision_or_dirty_checkout(self):
        with tempfile.TemporaryDirectory() as temp:
            with patch.object(adapter, "_run", side_effect=["wrong\n"]):
                with self.assertRaisesRegex(ValueError, "exactly pinned"):
                    adapter._verify_fixture(temp)
            with patch.object(adapter, "_run", side_effect=[adapter.CONTRACT_REV, " M scripts/qa_tfplan/plan_test.go\n"]):
                with self.assertRaisesRegex(ValueError, "exactly pinned"):
                    adapter._verify_fixture(temp)


class AdapterTest(unittest.TestCase):
    def test_partial_success_never_becomes_full_pass(self):
        scenario = SimpleNamespace(id="essentials-database-eviction-update")
        contract = SimpleNamespace(Evidence=lambda **kwargs: SimpleNamespace(**kwargs))
        instance = adapter.TerraformAdapter(contract, {scenario.id: scenario}, None, None, None, None)
        with patch.object(instance, "_exercise_existing_fixture", return_value="a" * 64):
            evidence = instance.execute(scenario)
        self.assertEqual(evidence.outcome, "unsupported")
        self.assertEqual(evidence.mode, "simulated")
        self.assertEqual(evidence.steps, ())
        self.assertEqual(evidence.cleanup, "not_started")

    def test_rejects_modified_scenario_and_does_not_run_lifecycle(self):
        lifecycle = SimpleNamespace(id="essentials-database-lifecycle", steps=("original",))
        contract = SimpleNamespace(Evidence=lambda **kwargs: SimpleNamespace(**kwargs))
        instance = adapter.TerraformAdapter(contract, {lifecycle.id: lifecycle}, None, None, None, None)
        with patch.object(instance, "_exercise_existing_fixture") as fixture:
            evidence = instance.execute(lifecycle)
            self.assertEqual(evidence.outcome, "unsupported")
            fixture.assert_not_called()
            with self.assertRaisesRegex(ValueError, "differs"):
                instance.execute(SimpleNamespace(id=lifecycle.id, steps=("modified",)))

    def test_rejects_mismatched_contract_bytes(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / "acceptance").mkdir()
            (root / "acceptance/contract.py").write_text("tampered")
            with patch.object(adapter, "_run", side_effect=[adapter.CONTRACT_REV, ""]):
                with self.assertRaisesRegex(ValueError, "hash mismatch"):
                    adapter._verify_fixture(temp)


if __name__ == "__main__":
    unittest.main()
