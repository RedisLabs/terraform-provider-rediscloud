package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestSubscriptionGracePeriodRequest(t *testing.T) {
	for _, name := range []string{"rediscloud_subscription", "rediscloud_active_active_subscription"} {
		t.Run(name, func(t *testing.T) {
			for _, activation := range []bool{false, true} {
				label := "grace period only"
				if activation {
					label = "activation includes keys"
				}
				t.Run(label, func(t *testing.T) {
					res := provider.NewSdkProvider("test")().ResourcesMap[name]
					key := map[string]interface{}{"resource_name": "projects/test/keys/key"}
					if name == "rediscloud_active_active_subscription" {
						key["region"] = "us-east1"
					}
					data := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
						"customer_managed_key_enabled":               true,
						"customer_managed_key_deletion_grace_period": "15-minutes",
						"customer_managed_key":                       []interface{}{key},
					})
					data.SetId("123")
					fake := testhelpers.NewFakeAPI()
					fake.Handle("GET /subscriptions/123", func(r *http.Request) (any, error) {
						status := "active"
						if activation {
							status = "encryption_key_pending"
						}
						return map[string]any{"id": 123, "status": status}, nil
					})
					var payload map[string]any
					calls := 0
					fake.Handle("PUT /subscriptions/123", func(r *http.Request) (any, error) {
						calls++
						if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
							return nil, err
						}
						if !activation {
							if _, exists := payload["customerManagedKeys"]; exists {
								return nil, testhelpers.StatusError{Code: http.StatusBadRequest, Body: map[string]any{"description": "customer keys must be omitted"}}
							}
						}
						// Stop here to also verify the resource propagates an API failure.
						return nil, testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "grace-period-update-denied"}}
					})
					diags := res.UpdateContext(context.Background(), data, testhelpers.NewAPIClient(t, fake))
					require.True(t, diags.HasError())
					require.Contains(t, diags[0].Summary, "grace-period-update-denied")
					fake.WithHandlersPaused(func() {
						require.Equal(t, 1, calls)
						require.Equal(t, "15-minutes", payload["deletionGracePeriod"])
						if activation {
							require.NotEmpty(t, payload["customerManagedKeys"])
						} else {
							require.Equal(t, map[string]any{"deletionGracePeriod": "15-minutes"}, payload)
						}
					})
				})
			}
			t.Run("unchanged grace period", func(t *testing.T) {
				res := provider.NewSdkProvider("test")().ResourcesMap[name]
				original := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
					"customer_managed_key_enabled":               true,
					"customer_managed_key_deletion_grace_period": "15-minutes",
				})
				original.SetId("123")
				data := res.Data(original.State())
				require.False(t, data.HasChange("customer_managed_key_deletion_grace_period"))
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				fake := testhelpers.NewFakeAPI()
				fake.Handle("GET /subscriptions/123", func(r *http.Request) (any, error) {
					cancel() // Stop the subsequent active waiter; no update should be sent.
					return map[string]any{"id": 123, "status": "active"}, nil
				})
				calls := 0
				fake.Handle("PUT /subscriptions/123", func(r *http.Request) (any, error) {
					calls++
					return nil, testhelpers.StatusError{Code: http.StatusBadRequest, Body: map[string]any{"description": "unexpected update"}}
				})
				diags := res.UpdateContext(ctx, data, testhelpers.NewAPIClient(t, fake))
				require.True(t, diags.HasError())
				fake.WithHandlersPaused(func() { require.Zero(t, calls) })
			})
		})
	}
}
