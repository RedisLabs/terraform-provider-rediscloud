package unit_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryStateWithoutTimeouts(t *testing.T) {
	for _, kind := range []string{"rediscloud_agent_memory", "rediscloud_agent_memory_api_key"} {
		t.Run(kind, func(t *testing.T) {
			server, err := testhelpers.ProtoV5ProviderFactories()["rediscloud"]()
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			schemas, err := server.GetProviderSchema(ctx, &tfprotov5.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			prior := map[string]any{"id": "store-1", "name": "existing", "database_id": 123}
			if kind == "rediscloud_agent_memory_api_key" {
				prior = map[string]any{"id": "key-1", "store_id": "store-1", "name": "runtime", "api_key": "FAKE-EXISTING-SECRET"}
			}
			// Omitted optional blocks decode as empty lists without changing resource identity.
			raw, err := json.Marshal(prior)
			if err != nil {
				t.Fatal(err)
			}
			upgraded, err := server.UpgradeResourceState(ctx, &tfprotov5.UpgradeResourceStateRequest{TypeName: kind, Version: 0, RawState: &tfprotov5.RawState{JSON: raw}})
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range upgraded.Diagnostics {
				if diagnostic.Severity == tfprotov5.DiagnosticSeverityError {
					t.Fatalf("existing state could not be decoded: %s", diagnostic.Summary)
				}
			}
			if upgraded.UpgradedState == nil {
				t.Fatal("existing state was discarded")
			}
			value, err := upgraded.UpgradedState.Unmarshal(schemas.ResourceSchemas[kind].ValueType())
			if err != nil {
				t.Fatal(err)
			}
			var attrs map[string]tftypes.Value
			if err := value.As(&attrs); err != nil {
				t.Fatal(err)
			}
			timeout, ok := attrs["timeouts"]
			if !ok {
				t.Fatal("timeouts missing from schema")
			}
			var blocks []tftypes.Value
			if err := timeout.As(&blocks); err != nil || len(blocks) != 0 {
				t.Error("omitted timeouts must decode as an empty block list")
			}
			for _, field := range []string{"id", "name", "store_id", "api_key"} {
				if expected, ok := prior[field]; ok {
					var actual string
					if err := attrs[field].As(&actual); err != nil || actual != expected {
						t.Errorf("existing %s was changed during state decoding", field)
					}
				}
			}
		})
	}
}
