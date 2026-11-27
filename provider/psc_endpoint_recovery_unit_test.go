package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// API failures must not be interpreted as proof that a PSC endpoint is absent.
// This uses the real provider handlers and SDK against a loopback HTTP server.
func TestPSCTransientReadAndDeleteKeepState(t *testing.T) {
	for _, tc := range []struct {
		name, id, path string
		resource       *schema.Resource
		operation      func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics
	}{
		{"pro/read", "101/202/303", "/subscriptions/101/private-service-connect/202", resourceRedisCloudPrivateServiceConnectEndpoint(), resourceRedisCloudPrivateServiceConnectEndpointRead},
		{"pro/delete", "101/202/303", "/subscriptions/101/private-service-connect/202", resourceRedisCloudPrivateServiceConnectEndpoint(), resourceRedisCloudPrivateServiceConnectEndpointDelete},
		{"aa/read", "101/4/202/303", "/subscriptions/101/regions/4/private-service-connect/202", resourceRedisCloudActiveActivePrivateServiceConnectEndpoint(), resourceRedisCloudActiveActivePrivateServiceConnectEndpointRead},
		{"aa/delete", "101/4/202/303", "/subscriptions/101/regions/4/private-service-connect/202", resourceRedisCloudActiveActivePrivateServiceConnectEndpoint(), resourceRedisCloudActiveActivePrivateServiceConnectEndpointDelete},
	} {
		for _, mode := range []string{"forbidden", "unavailable", "task-missing-resource"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Method != http.MethodGet {
						t.Errorf("unexpected mutation: %s %s", r.Method, r.URL.Path)
						w.WriteHeader(500)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case tc.path:
						switch mode {
						case "forbidden":
							w.WriteHeader(http.StatusForbidden)
							_, _ = fmt.Fprint(w, `{"error":"forbidden"}`)
						case "unavailable":
							w.WriteHeader(http.StatusServiceUnavailable)
							_, _ = fmt.Fprint(w, `{"error":"unavailable"}`)
						default:
							_, _ = fmt.Fprint(w, `{"taskId":"fixture-task"}`)
						}
					case "/tasks/fixture-task":
						_, _ = fmt.Fprint(w, `{"status":"processing-completed"}`)
					default:
						t.Errorf("unexpected path %s", r.URL.Path)
						w.WriteHeader(http.StatusNotFound)
					}
				}))
				defer server.Close()
				api, err := rediscloudapi.NewClient(rediscloudapi.BaseURL(server.URL), rediscloudapi.Auth("fake-key", "fake-secret"))
				if err != nil {
					t.Fatal(err)
				}
				d := schema.TestResourceDataRaw(t, tc.resource.Schema, nil)
				d.SetId(tc.id)
				diags := tc.operation(context.Background(), d, &client.ApiClient{Client: api})
				if !diags.HasError() {
					t.Fatalf("expected error for %s", mode)
				}
				if d.Id() != tc.id {
					t.Fatalf("lost endpoint identity: got %q want %q", d.Id(), tc.id)
				}
				if calls == 0 {
					t.Fatal("fixture was not called")
				}
			})
		}
	}
}
