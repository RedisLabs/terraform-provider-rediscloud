package pro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-cty/cty/msgpack"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// This exercises the SDK protocol import and subsequent real handler read,
// using no Terraform CLI or live service. Import contains no configuration.
func TestProDatabaseProtocolImportSourceIPs(t *testing.T) {
	for _, tc := range []struct {
		name     string
		observed []string
		expected []string
		public   bool
	}{
		{name: "custom", observed: []string{"192.0.2.0/24"}, expected: []string{"192.0.2.0/24"}, public: true},
		{name: "public_default", observed: []string{"0.0.0.0/0"}, expected: []string{"0.0.0.0/0"}, public: true},
		{name: "private_default_public_parent", observed: defaultPrivateIPRanges, expected: []string{"0.0.0.0/0"}, public: true},
		{name: "private_default_private_parent", observed: defaultPrivateIPRanges, expected: defaultPrivateIPRanges},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Api-Key") != "fake-key" || r.Header.Get("X-Api-Secret-Key") != "fake-secret" || r.Method != http.MethodGet {
					t.Error("unexpected credentials or mutation")
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v1/subscriptions/1/databases/42":
					reads++
					_ = json.NewEncoder(w).Encode(map[string]any{"databaseId": 42, "name": "fixture", "protocol": "redis", "status": "active", "security": map[string]any{"enableDefaultUser": true, "enableTls": false, "tlsClientAuthentication": false, "sourceIps": tc.observed}, "throughputMeasurement": map[string]any{"by": "operations-per-second", "value": 1000}, "clustering": map[string]any{"regexRules": []any{}}, "modules": []any{}, "alerts": []any{}})
				case "/v1/subscriptions/1/databases/42/tags":
					_ = json.NewEncoder(w).Encode(map[string]any{"tags": []any{}})
				case "/v1/subscriptions/1":
					_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "status": "active", "publicEndpointAccess": tc.public})
				default:
					t.Error("unexpected fake route")
					http.Error(w, "forbidden", http.StatusForbidden)
				}
			}))
			defer server.Close()
			sdk, err := rediscloudapi.NewClient(rediscloudapi.BaseURL(server.URL+"/v1"), rediscloudapi.Auth("fake-key", "fake-secret"))
			if err != nil {
				t.Fatal(err)
			}
			resource := ResourceRedisCloudProDatabase()
			original := resource.ReadContext
			var readCalls int
			resource.ReadContext = func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
				readCalls++
				if !d.GetRawConfig().IsNull() {
					t.Error("import refresh unexpectedly had raw configuration")
				}
				return original(ctx, d, meta)
			}
			provider := &schema.Provider{ResourcesMap: map[string]*schema.Resource{"rediscloud_subscription_database": resource}}
			provider.SetMeta(&client.ApiClient{Client: sdk, WaitDelayOverride: time.Millisecond, WaitPollIntervalOverride: time.Millisecond})
			protocol := schema.NewGRPCProviderServer(provider)
			imported, err := protocol.ImportResourceState(context.Background(), &tfprotov5.ImportResourceStateRequest{TypeName: "rediscloud_subscription_database", ID: "1/42"})
			if err != nil || imported == nil || len(imported.Diagnostics) != 0 || len(imported.ImportedResources) != 1 {
				t.Fatal("protocol import failed")
			}
			item := imported.ImportedResources[0]
			result, err := protocol.ReadResource(context.Background(), &tfprotov5.ReadResourceRequest{TypeName: "rediscloud_subscription_database", CurrentState: item.State, Private: item.Private})
			if err != nil || result == nil || len(result.Diagnostics) != 0 || result.NewState == nil || readCalls != 1 || reads != 1 {
				t.Fatal("protocol import refresh failed or did not read the expected fake database")
			}
			state, err := msgpack.Unmarshal(result.NewState.MsgPack, resource.CoreConfigSchema().ImpliedType())
			if err != nil {
				t.Fatal(err)
			}
			ips := state.GetAttr("source_ips")
			if ips.IsNull() || !ips.IsKnown() || ips.LengthInt() != len(tc.expected) {
				t.Fatal("unexpected imported source CIDR count")
			}
			for _, expected := range tc.expected {
				if !ips.HasElement(cty.StringVal(expected)).True() {
					t.Fatal("unexpected imported source CIDR")
				}
			}
		})
	}
}
