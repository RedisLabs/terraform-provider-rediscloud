import json
import os
import unittest
from types import SimpleNamespace
from unittest.mock import patch
from urllib.error import HTTPError
from urllib.request import Request, urlopen

from lifecycle_fixture import Fixture
from terraform_lifecycle_adapter import TerraformLifecycleAdapter


class FixtureTests(unittest.TestCase):
    def test_only_fake_auth_and_run_owned_paths_can_mutate(self):
        fixture = Fixture()
        self.addCleanup(fixture.close)
        request = Request(fixture.url + "/fixed/subscriptions", data=b'{"name":"fixture"}', method="POST", headers={"Content-Type": "application/json"})
        with self.assertRaises(HTTPError) as error:
            urlopen(request, timeout=3)
        self.assertEqual(error.exception.code, 403)
        self.assertIsNone(fixture.subscription)
        request = Request(fixture.url + "/fixed/subscriptions/999", method="DELETE", headers={"X-Api-Key": "fake", "X-Api-Secret-Key": "fake"})
        with self.assertRaises(HTTPError) as error:
            urlopen(request, timeout=3)
        self.assertEqual(error.exception.code, 404)
        self.assertIsNone(fixture.subscription)

    def test_exact_sub_and_db_lifecycle_and_refuse_parent_with_child(self):
        fixture = Fixture()
        self.addCleanup(fixture.close)
        headers = {"X-Api-Key": "fake", "X-Api-Secret-Key": "fake", "Content-Type": "application/json"}
        def write(method, path, body=None):
            req = Request(fixture.url + path, data=json.dumps(body).encode() if body is not None else None, method=method, headers=headers)
            with urlopen(req, timeout=3) as resp:
                return resp.status
        self.assertEqual(write("POST", "/fixed/subscriptions", {"name": "fixture"}), 200)
        self.assertEqual(write("POST", "/fixed/subscriptions/101/databases", {"name": "fixture-db"}), 200)
        with self.assertRaises(HTTPError):
            write("DELETE", "/fixed/subscriptions/101")
        self.assertEqual(fixture.read("/fixed/subscriptions/101")[0], 200)
        self.assertEqual(write("DELETE", "/fixed/subscriptions/101/databases/202"), 200)
        self.assertEqual(fixture.read("/fixed/subscriptions/101/databases/202")[0], 404)
        self.assertEqual(fixture.read("/fixed/subscriptions/101")[0], 200)
        self.assertEqual(write("DELETE", "/fixed/subscriptions/101"), 200)
        self.assertEqual(fixture.read("/fixed/subscriptions/101")[0], 404)
        self.assertEqual(fixture.read("/fixed/subscriptions")[1], {"subscriptions": []})


class ReceiptTests(unittest.TestCase):
    def test_local_observations_cannot_claim_full_scenario(self):
        scenario = SimpleNamespace(id="test", steps=("first", "second"))
        contract = SimpleNamespace(Evidence=lambda **values: SimpleNamespace(**values), StepReceipt=lambda **values: SimpleNamespace(**values))
        adapter = TerraformLifecycleAdapter(contract, {"test": scenario}, None, None, None, None)
        def run_fixture(callback):
            callback("fake-tf", "fake-plugins")
            return "a" * 64
        with patch.dict(os.environ, {}, clear=True), patch.object(adapter, "_exercise_existing_fixture", side_effect=run_fixture), patch("terraform_lifecycle.run", return_value=list(scenario.steps)):
            evidence = adapter.execute(scenario)
        self.assertEqual(evidence.outcome, "unsupported")
        self.assertEqual(evidence.mode, "simulated")
        self.assertEqual(evidence.steps, ())

    def test_hosted_claim_is_simulated_and_receipts_use_actions_console(self):
        scenario = SimpleNamespace(id="test", steps=("first", "second"))
        contract = SimpleNamespace(Evidence=lambda **values: SimpleNamespace(**values), StepReceipt=lambda **values: SimpleNamespace(**values))
        adapter = TerraformLifecycleAdapter(contract, {"test": scenario}, None, None, None, None)
        def run_fixture(callback):
            callback("fake-tf", "fake-plugins")
            return "a" * 64
        env = {"GITHUB_ACTIONS": "true", "GITHUB_REPOSITORY": "RedisLabs/terraform-provider-rediscloud", "GITHUB_RUN_ID": "123"}
        with patch.dict(os.environ, env, clear=True), patch.object(adapter, "_exercise_existing_fixture", side_effect=run_fixture), patch("terraform_lifecycle.run", return_value=list(scenario.steps)):
            evidence = adapter.execute(scenario)
        self.assertEqual(evidence.mode, "simulated")
        self.assertEqual(evidence.outcome, "pass")
        self.assertEqual([step.step for step in evidence.steps], list(scenario.steps))
        self.assertTrue(all(step.evidence_url == "https://github.com/RedisLabs/terraform-provider-rediscloud/actions/runs/123" for step in evidence.steps))
