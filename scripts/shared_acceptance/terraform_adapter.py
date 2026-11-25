"""Simulated, partial adapter for the provisional shared acceptance contract.

Runs only a pinned loopback fixture; it cannot run a live qualification or
claim a full shared scenario pass. See docs/design/shared-acceptance-prototype.md.
"""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
import platform
from pathlib import Path
import subprocess
import sys
import tempfile

CONTRACT_REV = "7ab7fcbb7f70c01dbf08d9df3e40f072ec24b20e"
SOURCE_REV = "043ad6617389634044d17e4c51dd4203a3ffb134"
HASHES = {
    "acceptance/contract.py": "d886f1c07d769cef790677a4d4edd52355a15feffbc5c64cd0fa1017b89a1fcc",
    "acceptance/scenarios.json": "66f926d8c9a26acb6b2952b68280a1f82e71e98d2f463ff32862505b3989865a",
}


def _run(args, *, cwd, env=None):
    try:
        return subprocess.run(args, cwd=cwd, env=env, check=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT).stdout
    except subprocess.CalledProcessError as exc:
        raise RuntimeError(f"fixture command failed: {exc.output}") from exc


def _digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def _verify_fixture(root):
    root = Path(root).resolve(strict=True)
    revision = _run(["git", "rev-parse", "HEAD"], cwd=root).strip()
    if revision != CONTRACT_REV or _run(["git", "status", "--porcelain"], cwd=root).strip():
        raise ValueError("fixture worktree must be clean and exactly pinned")
    for name, digest in HASHES.items():
        if _digest(root / name) != digest:
            raise ValueError(f"shared contract hash mismatch: {name}")
    return root


def _load_contract(root):
    sys.dont_write_bytecode = True
    spec = importlib.util.spec_from_file_location("shared_contract_pinned", root / "acceptance/contract.py")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def run_partial(source_repo, fixture_repo, go_bin, terraform_bin, scenario_id):
    if sys.platform != "linux" or platform.machine() not in {"x86_64", "amd64"}:
        raise ValueError("the pinned fixture supports linux/amd64 only")
    source_repo = Path(source_repo).resolve(strict=True)
    fixture_repo = _verify_fixture(fixture_repo)
    go_bin = str(Path(go_bin).resolve(strict=True))
    terraform_bin = str(Path(terraform_bin).resolve(strict=True))
    contract = _load_contract(fixture_repo)
    scenarios = contract.load_scenarios(fixture_repo / "acceptance/scenarios.json")
    scenario = scenarios[scenario_id]
    # Adapter.execute below is the shared protocol; the separate report carries
    # local provenance and never substitutes for per-step evidence receipts.
    adapter = TerraformAdapter(contract, scenarios, source_repo, fixture_repo, go_bin, terraform_bin)
    evidence = adapter.execute(scenario)
    return {"provider": evidence.provider, "scenario_id": evidence.scenario_id, "mode": evidence.mode, "source_revision": evidence.source_revision, "artifact_digest": evidence.artifact_digest, "fixture_revision": CONTRACT_REV, "terraform_binary_sha256": _digest(terraform_bin), "observation": adapter.observation, "classification": contract.classify_claim(scenario, evidence), "limitation": "no full lifecycle, live Cloud, or cleanup qualification"}


class TerraformAdapter:
    """Implementation of the provisional Adapter protocol for partial simulation."""

    def __init__(self, contract, scenarios, source_repo, fixture_repo, go_bin, terraform_bin):
        self.contract = contract
        self.scenarios = scenarios
        self.source_repo = source_repo
        self.fixture_repo = fixture_repo
        self.go_bin = go_bin
        self.terraform_bin = terraform_bin
        self.observation = "none"

    def execute(self, scenario):
        if self.scenarios.get(scenario.id) != scenario:
            raise ValueError("scenario differs from pinned contract")
        if scenario.id != "essentials-database-eviction-update":
            return self.contract.Evidence(provider="terraform", scenario_id=scenario.id, mode="simulated", source_revision=SOURCE_REV, artifact_digest="", run_url="", evidence_url="", steps=(), outcome="unsupported", cleanup="not_started", direct_cloud_recovery=False)
        digest = self._exercise_existing_fixture()
        self.observation = "loopback imported database: eviction update and stable identity observed"
        return self.contract.Evidence(provider="terraform", scenario_id=scenario.id, mode="simulated", source_revision=SOURCE_REV, artifact_digest="sha256:" + digest, run_url="", evidence_url="", steps=(), outcome="unsupported", cleanup="not_started", direct_cloud_recovery=False)

    def _exercise_existing_fixture(self):
        source_repo, fixture_repo, go_bin, terraform_bin = self.source_repo, self.fixture_repo, self.go_bin, self.terraform_bin
        # Never pass ambient cloud or Terraform configuration to the fixture.
        allowed = {"PATH", "HOME", "GOPATH", "GOCACHE", "GOMODCACHE", "TMPDIR", "SYSTEMROOT"}
        env = {k: v for k, v in os.environ.items() if k in allowed}
        env.update({"GOPROXY": "off", "GOSUMDB": "off", "GOTOOLCHAIN": "local"})
        with tempfile.TemporaryDirectory(prefix="tf-shared-fixture-") as temp:
            temp = Path(temp)
            source = temp / "source"
            source.mkdir()
            archive = subprocess.run(["git", "archive", SOURCE_REV], cwd=source_repo, check=True, stdout=subprocess.PIPE).stdout
            subprocess.run(["tar", "-x", "-C", str(source)], input=archive, check=True)
            plugins = temp / "plugins"
            # This fixture layout is a compatibility label, not a release claim.
            binary = plugins / "registry.terraform.io/redislabs/rediscloud/2.20.0/linux_amd64/terraform-provider-rediscloud_v2.20.0"
            binary.parent.mkdir(parents=True)
            _run([go_bin, "build", "-trimpath", "-o", str(binary), "."], cwd=source, env=env)
            digest = _digest(binary)
            env.update({"QA_TEST_TERRAFORM": terraform_bin, "QA_TEST_PROVIDER_DIR": str(plugins)})
            output = _run([go_bin, "test", "./scripts/qa_tfplan", "-run", "^TestEssentialsRefreshAndPlan$", "-count=1", "-v"], cwd=fixture_repo, env=env)
            if "--- PASS: TestEssentialsRefreshAndPlan" not in output:
                raise ValueError("fixture did not run and pass")
        return digest


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--source-repo", required=True)
    p.add_argument("--fixture-repo", required=True)
    p.add_argument("--go", required=True)
    p.add_argument("--terraform", required=True)
    p.add_argument("--scenario", default="essentials-database-eviction-update")
    args = p.parse_args()
    print(json.dumps(run_partial(args.source_repo, args.fixture_repo, args.go, args.terraform, args.scenario), sort_keys=True))


if __name__ == "__main__":
    main()
