"""Shared-scenario adapter for the loopback-only Terraform lifecycle fixture."""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path

import terraform_adapter as base
import terraform_lifecycle


class TerraformLifecycleAdapter(base.TerraformAdapter):
    """Produce simulated receipts; never execute against a real Cloud account."""

    def execute(self, scenario):
        if self.scenarios.get(scenario.id) != scenario:
            raise ValueError("scenario differs from pinned contract")
        observations = []

        def exercise(terraform_bin, plugins):
            observations.extend(terraform_lifecycle.run(scenario, terraform_bin, plugins))

        digest = self._exercise_existing_fixture(exercise)
        self.observation = ",".join(observations)
        # A hosted Actions console is a real evidence location. Outside that
        # environment do not manufacture links or claim a complete result.
        run_id = os.environ.get("GITHUB_RUN_ID", "")
        if os.environ.get("GITHUB_ACTIONS") != "true" or os.environ.get("GITHUB_REPOSITORY") != "RedisLabs/terraform-provider-rediscloud" or not run_id.isascii() or not run_id.isdigit() or int(run_id) == 0:
            return self.contract.Evidence(provider="terraform", scenario_id=scenario.id, mode="simulated", source_revision=base.SOURCE_REV, artifact_digest="sha256:" + digest, run_url="", evidence_url="", steps=(), outcome="unsupported", cleanup="not_started", direct_cloud_recovery=False)
        run_url = f"https://github.com/RedisLabs/terraform-provider-rediscloud/actions/runs/{run_id}"
        receipts = tuple(self.contract.StepReceipt(step=step, outcome="passed", evidence_url=run_url) for step in observations)
        return self.contract.Evidence(provider="terraform", scenario_id=scenario.id, mode="simulated", source_revision=base.SOURCE_REV, artifact_digest="sha256:" + digest, run_url=run_url, evidence_url=run_url, steps=receipts, outcome="pass", cleanup="clean", direct_cloud_recovery=False)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source-repo", required=True)
    parser.add_argument("--fixture-repo", required=True)
    parser.add_argument("--go", required=True)
    parser.add_argument("--terraform", required=True)
    parser.add_argument("--scenario", required=True)
    args = parser.parse_args()
    source = Path(args.source_repo).resolve(strict=True)
    fixture = base._verify_fixture(args.fixture_repo)
    contract = base._load_contract(fixture)
    scenarios = contract.load_scenarios(fixture / "acceptance/scenarios.json")
    scenario = scenarios[args.scenario]
    adapter = TerraformLifecycleAdapter(contract, scenarios, source, fixture, str(Path(args.go).resolve(strict=True)), str(Path(args.terraform).resolve(strict=True)))
    result = adapter.execute(scenario)
    print(json.dumps({"provider": result.provider, "scenario": result.scenario_id, "mode": result.mode, "classification": contract.classify_claim(scenario, result), "source_revision": result.source_revision, "artifact_digest": result.artifact_digest, "fixture_revision": base.CONTRACT_REV, "run_url": result.run_url, "steps": adapter.observation.split(","), "limitation": "simulated loopback only; no live Cloud or account lock evidence"}, sort_keys=True))


if __name__ == "__main__":
    main()
