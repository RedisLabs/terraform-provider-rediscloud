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

// The fake rejects the create after capturing the request. This exercises the
// production handler and SDK serializer without creating a Cloud resource.
func TestUnitProDatabaseCreateSourceIPs(t *testing.T) {
	for _, tc := range []struct {
		name string
		ips  []any
	}{
		{name: "explicit", ips: []any{"192.0.2.0/24", "198.51.100.0/24"}},
		{name: "omitted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			var payload map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Api-Key") != "fake-key" || r.Header.Get("X-Api-Secret-Key") != "fake-secret" {
					http.Error(w, "unexpected credentials", http.StatusForbidden)
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == "/v1/subscriptions/1" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"id":1,"status":"active"}`))
					return
				}
				if r.Method == http.MethodPost && r.URL.Path == "/v1/subscriptions/1/databases" {
					requests++
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Errorf("decode create: %v", err)
					}
					http.Error(w, "intentional rejection", http.StatusBadRequest)
					return
				}
				t.Errorf("unexpected fake route: %s %s", r.Method, r.URL.Path)
				http.Error(w, "unexpected route", http.StatusForbidden)
			}))
			defer server.Close()
			sdk, err := rediscloudapi.NewClient(rediscloudapi.BaseURL(server.URL+"/v1"), rediscloudapi.Auth("fake-key", "fake-secret"))
			if err != nil {
				t.Fatal(err)
			}
			values := map[string]any{"subscription_id": 1, "name": "fixture"}
			if tc.ips != nil {
				values["source_ips"] = tc.ips
			}
			d := schema.TestResourceDataRaw(t, ResourceRedisCloudProDatabase().Schema, values)
			api := &client.ApiClient{Client: sdk, WaitDelayOverride: time.Millisecond, WaitPollIntervalOverride: time.Millisecond}
			result := resourceRedisCloudProDatabaseCreate(context.Background(), d, api)
			if !result.HasError() || requests != 1 {
				t.Fatalf("expected one rejected create; got %d requests", requests)
			}
			actual, exists := payload["sourceIp"]
			if tc.ips == nil {
				if exists {
					t.Fatal("omitted source_ips unexpectedly sent")
				}
				return
			}
			items, ok := actual.([]any)
			seen := make(map[string]bool)
			for _, item := range items {
				value, valid := item.(string)
				if valid {
					seen[value] = true
				}
			}
			if !ok || len(items) != 2 || !seen["192.0.2.0/24"] || !seen["198.51.100.0/24"] {
				t.Fatalf("incorrect create sourceIp: %v", actual)
			}
		})
	}
}
