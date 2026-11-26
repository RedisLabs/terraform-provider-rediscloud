package pro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestUnitProDatabaseReadObservedSourceIPs(t *testing.T) {
	for _, tc := range []struct {
		name        string
		field       any
		include     bool
		security    string
		want        []string
		wantFailure bool
	}{
		{name: "missing_security", security: "missing", wantFailure: true},
		{name: "null_security", security: "null", wantFailure: true},
		{name: "missing", wantFailure: true},
		{name: "null", include: true, wantFailure: true},
		{name: "null_element", include: true, field: []any{nil}, wantFailure: true},
		{name: "empty", include: true, field: []any{}, want: []string{}},
		{name: "public_drift", include: true, field: []any{"0.0.0.0/0"}, want: []string{"0.0.0.0/0"}},
		{name: "custom_drift", include: true, field: []any{"198.51.100.0/24"}, want: []string{"198.51.100.0/24"}},
		{name: "unchanged", include: true, field: []any{"192.0.2.0/24"}, want: []string{"192.0.2.0/24"}},
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
					security := map[string]any{"enableDefaultUser": true, "enableTls": false, "tlsClientAuthentication": false}
					if tc.include {
						security["sourceIps"] = tc.field
					}
					response := map[string]any{"databaseId": 42, "name": "changed", "protocol": "redis", "status": "active", "throughputMeasurement": map[string]any{"by": "operations-per-second", "value": 1000}, "clustering": map[string]any{"regexRules": []any{}}, "modules": []any{}, "alerts": []any{}}
					if tc.security == "null" {
						response["security"] = nil
					} else if tc.security != "missing" {
						response["security"] = security
					}
					_ = json.NewEncoder(w).Encode(response)
				case "/v1/subscriptions/1/databases/42/tags":
					_ = json.NewEncoder(w).Encode(map[string]any{"tags": []any{}})
				default:
					t.Errorf("unexpected fake route: %s", r.URL.Path)
					http.Error(w, "forbidden", http.StatusForbidden)
				}
			}))
			defer server.Close()
			sdk, err := rediscloudapi.NewClient(rediscloudapi.BaseURL(server.URL+"/v1"), rediscloudapi.Auth("fake-key", "fake-secret"))
			if err != nil {
				t.Fatal(err)
			}
			d := schema.TestResourceDataRaw(t, ResourceRedisCloudProDatabase().Schema, map[string]any{"subscription_id": 1, "name": "fixture", "source_ips": []any{"192.0.2.0/24"}})
			d.SetId("1/42")
			api := &client.ApiClient{Client: sdk, WaitDelayOverride: time.Millisecond, WaitPollIntervalOverride: time.Millisecond}
			result := resourceRedisCloudProDatabaseRead(context.Background(), d, api)
			if reads != 1 || result.HasError() != tc.wantFailure {
				t.Fatalf("unexpected fake read outcome: reads=%d error=%t", reads, result.HasError())
			}
			if tc.wantFailure {
				if d.Get("name") != "fixture" || d.Get("db_id") != 0 {
					t.Fatal("invalid security observation changed state before failure")
				}
				return
			}
			observed := d.Get("source_ips").(*schema.Set).List()
			if len(observed) != len(tc.want) {
				t.Fatalf("wrong source CIDR count: %d", len(observed))
			}
			for _, expected := range tc.want {
				if !d.Get("source_ips").(*schema.Set).Contains(expected) {
					t.Fatalf("expected observed source CIDR %s", expected)
				}
			}
		})
	}
}
