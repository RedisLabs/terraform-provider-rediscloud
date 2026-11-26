package pro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
		name       string
		ips        []any
		public     *bool
		lookupFail bool
		want       []string
		wantCreate bool
		wantError  string
	}{
		{name: "custom_private_parent", ips: []any{"192.0.2.0/24", "198.51.100.0/24"}, public: boolPtr(false), want: []string{"192.0.2.0/24", "198.51.100.0/24"}, wantCreate: true},
		{name: "custom_public_parent", ips: []any{"192.0.2.0/24"}, public: boolPtr(true), want: []string{"192.0.2.0/24"}, wantCreate: true},
		{name: "omitted_private_parent", public: boolPtr(false), wantCreate: true},
		{name: "omitted_public_parent", public: boolPtr(true), wantCreate: true},
		{name: "public_default_private_parent", ips: []any{"0.0.0.0/0"}, public: boolPtr(false), want: defaultPrivateIPRanges, wantCreate: true},
		{name: "public_default_public_parent", ips: []any{"0.0.0.0/0"}, public: boolPtr(true), want: []string{"0.0.0.0/0"}, wantCreate: true},
		{name: "private_default_public_parent", ips: []any{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10"}, public: boolPtr(true), wantError: "conflict with public endpoint access"},
		{name: "private_default_private_parent", ips: []any{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10"}, public: boolPtr(false), want: defaultPrivateIPRanges, wantCreate: true},
		{name: "default_unknown_policy", ips: []any{"0.0.0.0/0"}, wantError: "cannot determine subscription public endpoint access"},
		{name: "default_lookup_failure", ips: []any{"0.0.0.0/0"}, public: boolPtr(false), lookupFail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			subscriptionReads := 0
			var payload map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Api-Key") != "fake-key" || r.Header.Get("X-Api-Secret-Key") != "fake-secret" {
					http.Error(w, "unexpected credentials", http.StatusForbidden)
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == "/v1/subscriptions/1" {
					subscriptionReads++
					if tc.lookupFail && subscriptionReads == 2 {
						http.Error(w, "injected lookup failure", http.StatusForbidden)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					response := map[string]any{"id": 1, "status": "active"}
					if tc.public != nil {
						response["publicEndpointAccess"] = *tc.public
					}
					_ = json.NewEncoder(w).Encode(response)
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
			expectedRequests := 0
			if tc.wantCreate {
				expectedRequests = 1
			}
			if !result.HasError() || requests != expectedRequests {
				t.Fatalf("expected %d create requests; got %d", expectedRequests, requests)
			}
			if !tc.wantCreate {
				if subscriptionReads != 2 {
					t.Fatalf("expected policy lookup before refusing create; got %d reads", subscriptionReads)
				}
				if tc.wantError != "" && (len(result) == 0 || !strings.Contains(result[0].Summary, tc.wantError)) {
					t.Fatal("missing expected policy refusal diagnostic")
				}
				return
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
			valid := ok && len(items) == len(tc.want) && len(seen) == len(tc.want)
			for _, expected := range tc.want {
				valid = valid && seen[expected]
			}
			if !valid {
				t.Fatalf("incorrect create sourceIp: %v", actual)
			}
		})
	}
}

func boolPtr(value bool) *bool { return &value }
