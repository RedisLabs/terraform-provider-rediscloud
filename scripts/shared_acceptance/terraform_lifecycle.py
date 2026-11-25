"""Execute shared Essentials steps with real Terraform against a loopback fake."""
from __future__ import annotations

import json
import os
from pathlib import Path
import subprocess
import tempfile

from lifecycle_fixture import Fixture


def run(scenario, terraform_bin, plugin_dir):
    fixture = Fixture()
    observations = []
    outputs = {}
    try:
        with tempfile.TemporaryDirectory(prefix="terraform-lifecycle-") as temp:
            root = Path(temp)
            config_path = root / "main.tf.json"
            (root / "empty.tfrc").write_text("")
            allowed = {"PATH", "HOME", "TMPDIR", "SYSTEMROOT"}
            env = {k: v for k, v in os.environ.items() if k in allowed}
            env.update({"TF_CLI_CONFIG_FILE": str(root / "empty.tfrc"), "TF_IN_AUTOMATION": "1", "REDISCLOUD_URL": fixture.url, "REDISCLOUD_ACCESS_KEY": "fake", "REDISCLOUD_SECRET_KEY": "fake"})

            def tf(*args):
                try:
                    result = subprocess.run([terraform_bin, *args], cwd=root, env=env, check=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, timeout=240)
                except subprocess.CalledProcessError as exc:
                    raise RuntimeError(f"Terraform command failed: {exc.output}") from exc
                return result.stdout

            def config(sub, db, eviction="volatile-lru"):
                resources = {}
                output = {}
                if sub:
                    resources["rediscloud_essentials_subscription"] = {"test": {"name": "fixture-sub", "plan_id": 1}}
                    output["sub_id"] = {"value": "${rediscloud_essentials_subscription.test.id}"}
                if db:
                    resources["rediscloud_essentials_database"] = {"test": {"subscription_id": "${rediscloud_essentials_subscription.test.id}", "name": "fixture-db", "data_eviction": eviction, "data_persistence": "none", "replication": False}}
                    output["db_id"] = {"value": "${rediscloud_essentials_database.test.id}"}
                document = {"terraform": {"required_providers": {"rediscloud": {"source": "RedisLabs/rediscloud", "version": "2.20.0"}}}, "provider": {"rediscloud": {"api_key": "fake", "secret_key": "fake", "url": fixture.url}}}
                if resources:
                    document["resource"] = resources
                if output:
                    document["output"] = output
                config_path.write_text(json.dumps(document))

            def apply(sub, db, eviction="volatile-lru"):
                config(sub, db, eviction)
                tf("apply", "-auto-approve", "-input=false", "-no-color")

            def observe(expected, condition):
                if not condition:
                    raise AssertionError(f"shared step failed: {expected}")
                observations.append(expected)
                print(json.dumps({"simulated_step": expected, "result": "passed"}), flush=True)

            # This fixture owns the entire in-memory account; it cannot authorize
            # a live account or stand in for the shared production lock.
            status, inventory = fixture.read("/fixed/subscriptions")
            observe("admission_and_clean_baseline", status == 200 and inventory == {"subscriptions": []})
            config(True, False)
            tf("init", "-no-color", "-input=false", "-plugin-dir=" + str(plugin_dir))
            apply(True, False)
            outputs = json.loads(tf("output", "-json"))
            observe("create_subscription", outputs["sub_id"]["value"] == "101")
            status, sub = fixture.read("/fixed/subscriptions/101")
            observe("read_subscription_active", status == 200 and sub.get("id") == 101 and sub.get("name") == "fixture-sub" and sub.get("status") == "active")
            apply(True, True)
            outputs = json.loads(tf("output", "-json"))
            observe("create_database", outputs["db_id"]["value"] == "101/202")
            status, database = fixture.read("/fixed/subscriptions/101/databases/202")
            observe("read_database_active", status == 200 and database.get("databaseId") == 202 and database.get("name") == "fixture-db" and database.get("status") == "active")
            if "update_eviction" in scenario.steps:
                previous_updates = fixture.events.count(("PUT", "/v1/fixed/subscriptions/101/databases/202"))
                apply(True, True, "allkeys-lru")
                observe("update_eviction", fixture.events.count(("PUT", "/v1/fixed/subscriptions/101/databases/202")) == previous_updates + 1)
                after = json.loads(tf("output", "-json"))
                status, database = fixture.read("/fixed/subscriptions/101/databases/202")
                observe("read_updated_eviction_same_identity", after["db_id"]["value"] == outputs["db_id"]["value"] and after["sub_id"]["value"] == outputs["sub_id"]["value"] and status == 200 and database.get("dataEvictionPolicy") == "allkeys-lru")
            apply(True, False)
            observe("delete_database", ("DELETE", "/v1/fixed/subscriptions/101/databases/202") in fixture.events)
            db_status, _ = fixture.read("/fixed/subscriptions/101/databases/202")
            sub_status, sub = fixture.read("/fixed/subscriptions/101")
            observe("read_database_absent_parent_present", db_status == 404 and sub_status == 200 and sub.get("id") == 101)
            apply(False, False)
            observe("delete_subscription", ("DELETE", "/v1/fixed/subscriptions/101") in fixture.events)
            sub_status, _ = fixture.read("/fixed/subscriptions/101")
            observe("read_subscription_absent", sub_status == 404)
            status, inventory = fixture.read("/fixed/subscriptions")
            observe("complete_clean_inventory", status == 200 and inventory == {"subscriptions": []} and fixture.database is None and not fixture.errors)
            if tuple(observations) != scenario.steps:
                raise AssertionError("fixture steps do not match the pinned shared scenario")
            print(json.dumps({"simulated_cloud_writes": [event for event in fixture.events if event[0] != "GET"]}), flush=True)
            return observations
    finally:
        fixture.close()
