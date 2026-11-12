package provider_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func cmkTestConfig(name string) map[string]interface{} {
	key := map[string]interface{}{"resource_name": "projects/test/keys/key"}
	if name == "rediscloud_active_active_subscription" {
		key["region"] = "us-east1"
	}
	return map[string]interface{}{
		"customer_managed_key_enabled": true,
		"customer_managed_key":         []interface{}{key},
	}
}

func TestSubscriptionCmkReadWhilePending(t *testing.T) {
	for _, name := range []string{"rediscloud_subscription", "rediscloud_active_active_subscription"} {
		for _, status := range []string{"encryption_key_pending", "active"} {
			t.Run(name+"/"+status, func(t *testing.T) {
				res := provider.NewSdkProvider("test")().ResourcesMap[name]
				original := schema.TestResourceDataRaw(t, res.Schema, cmkTestConfig(name))
				original.SetId("123")
				data := res.Data(original.State())
				require.Len(t, data.Get("customer_managed_key").([]interface{}), 1)

				fake := testhelpers.NewFakeAPI()
				fake.Handle("GET /subscriptions/123", func(r *http.Request) (any, error) {
					return map[string]any{
						"id":                              123,
						"status":                          status,
						"persistentStorageEncryptionType": "customer-managed-key",
						// Cloud account 1 skips the Pro allowlist lookup.
						"cloudDetails": []map[string]any{{"provider": "GCP", "cloudAccountId": 1}},
					}, nil
				})
				fake.Handle("GET /subscriptions/{id}/maintenance-windows", func(r *http.Request) (any, error) {
					return map[string]any{"mode": "automatic"}, nil
				})
				fake.Handle("GET /subscriptions/{id}/pricing", func(r *http.Request) (any, error) {
					return map[string]any{"pricing": []any{}}, nil
				})

				diags := res.ReadContext(context.Background(), data, testhelpers.NewAPIClient(t, fake))
				require.False(t, diags.HasError(), "%v", diags)
				keys := data.Get("customer_managed_key").([]interface{})
				if status == "encryption_key_pending" {
					// Configured keys must show as a change so the next apply sends them.
					require.Empty(t, keys)
				} else {
					// Once active, the applied keys stay in state, so there is no perpetual diff.
					require.Len(t, keys, 1)
				}
			})
		}
	}
}

func TestSubscriptionCmkFailedActivationKeepsPriorState(t *testing.T) {
	for _, name := range []string{"rediscloud_subscription", "rediscloud_active_active_subscription"} {
		for _, failure := range []string{"update", "wait"} {
			t.Run(name+"/"+failure, func(t *testing.T) {
				res := provider.NewSdkProvider("test")().ResourcesMap[name]
				data := schema.TestResourceDataRaw(t, res.Schema, cmkTestConfig(name))
				data.SetId("123")

				fake := testhelpers.NewFakeAPI()
				// The update reads the status, then the waiter sees the API still reporting
				// encryption_key_pending before the subscription fails.
				states := []string{"encryption_key_pending", "encryption_key_pending", "error"}
				gets := 0
				fake.Handle("GET /subscriptions/123", func(r *http.Request) (any, error) {
					status := states[min(gets, len(states)-1)]
					gets++
					return map[string]any{"id": 123, "status": status}, nil
				})
				fake.Handle("PUT /subscriptions/123", func(r *http.Request) (any, error) {
					if failure == "update" {
						return nil, testhelpers.StatusError{Code: http.StatusBadRequest, Body: map[string]any{"description": "key not accessible"}}
					}
					return fake.RegisterTask("subscriptionUpdate", 123), nil
				})

				diags := res.UpdateContext(context.Background(), data, testhelpers.NewAPIClient(t, fake))
				require.True(t, diags.HasError())
				if failure == "wait" {
					require.Contains(t, diags[0].Summary, "unexpected state 'error'")
				}
				// The planned keys must not be persisted, so the next apply retries the activation.
				state := data.State()
				require.NotNil(t, state)
				require.Equal(t, "0", state.Attributes["customer_managed_key.#"])
				require.NotContains(t, state.Attributes, "customer_managed_key.0.resource_name")
			})
		}
	}
}
