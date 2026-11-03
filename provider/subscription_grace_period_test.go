package provider_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

type gracePeriodSubscription struct {
	resourceName       string
	status             string
	gracePeriod        string
	gracePeriodChanged bool
	configuredKey      string
}

func TestSubscriptionGracePeriodRequest(t *testing.T) {
	tests := []struct {
		name         string
		given        gracePeriodSubscription
		wantRequests []map[string]any
		wantError    string
	}{
		{
			name: "Pro/grace period change omits configured keys",
			given: gracePeriodSubscription{
				resourceName:       "rediscloud_subscription",
				status:             "active",
				gracePeriod:        "15-minutes",
				gracePeriodChanged: true,
				configuredKey:      "projects/test/keys/key",
			},
			wantRequests: []map[string]any{
				{"deletionGracePeriod": "15-minutes"},
			},
			wantError: "grace-period-update-denied",
		},
		{
			name: "Pro/CMK activation sends grace period and configured keys",
			given: gracePeriodSubscription{
				resourceName:       "rediscloud_subscription",
				status:             "encryption_key_pending",
				gracePeriod:        "15-minutes",
				gracePeriodChanged: true,
				configuredKey:      "projects/test/keys/key",
			},
			wantRequests: []map[string]any{
				{
					"deletionGracePeriod": "15-minutes",
					"customerManagedKeys": []any{
						map[string]any{"resourceName": "projects/test/keys/key"},
					},
				},
			},
			wantError: "grace-period-update-denied",
		},
		{
			name: "Pro/unchanged grace period sends no update",
			given: gracePeriodSubscription{
				resourceName:       "rediscloud_subscription",
				status:             "active",
				gracePeriod:        "15-minutes",
				gracePeriodChanged: false,
			},
			wantRequests: nil,
			wantError:    "subscription-polling-stopped",
		},
		{
			name: "Active-Active/grace period change omits configured keys",
			given: gracePeriodSubscription{
				resourceName:       "rediscloud_active_active_subscription",
				status:             "active",
				gracePeriod:        "15-minutes",
				gracePeriodChanged: true,
				configuredKey:      "projects/test/keys/key",
			},
			wantRequests: []map[string]any{
				{"deletionGracePeriod": "15-minutes"},
			},
			wantError: "grace-period-update-denied",
		},
		{
			name: "Active-Active/CMK activation sends grace period and configured keys",
			given: gracePeriodSubscription{
				resourceName:       "rediscloud_active_active_subscription",
				status:             "encryption_key_pending",
				gracePeriod:        "15-minutes",
				gracePeriodChanged: true,
				configuredKey:      "projects/test/keys/key",
			},
			wantRequests: []map[string]any{
				{
					"deletionGracePeriod": "15-minutes",
					"customerManagedKeys": []any{
						map[string]any{"resourceName": "projects/test/keys/key", "region": "us-east1"},
					},
				},
			},
			wantError: "grace-period-update-denied",
		},
		{
			name: "Active-Active/unchanged grace period sends no update",
			given: gracePeriodSubscription{
				resourceName:       "rediscloud_active_active_subscription",
				status:             "active",
				gracePeriod:        "15-minutes",
				gracePeriodChanged: false,
			},
			wantRequests: nil,
			wantError:    "subscription-polling-stopped",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resource, data := newGracePeriodResourceData(t, test.given)
			api := newGracePeriodTestAPI(test.given.status)

			diags := resource.UpdateContext(t.Context(), data, testhelpers.NewAPIClient(t, api.fake))

			require.True(t, diags.HasError())
			require.Contains(t, diags[0].Summary, test.wantError)
			require.Equal(t, test.wantRequests, api.updateRequests())
		})
	}
}

// newGracePeriodResourceData prepares the configured CMK values for a direct Update call.
// The returned data makes Terraform's change detection match given.gracePeriodChanged.
func newGracePeriodResourceData(t *testing.T, given gracePeriodSubscription) (*schema.Resource, *schema.ResourceData) {
	t.Helper()

	resource := provider.NewSdkProvider("test")().ResourcesMap[given.resourceName]
	config := map[string]any{
		"customer_managed_key_enabled":               true,
		"customer_managed_key_deletion_grace_period": given.gracePeriod,
	}
	if given.configuredKey != "" {
		key := map[string]any{"resource_name": given.configuredKey}
		if given.resourceName == "rediscloud_active_active_subscription" {
			key["region"] = "us-east1"
		}
		config["customer_managed_key"] = []any{key}
	}
	data := schema.TestResourceDataRaw(t, resource.Schema, config)
	data.SetId("123")
	if !given.gracePeriodChanged {
		data = resource.Data(data.State())
	}
	require.Equal(t, given.gracePeriodChanged, data.HasChange("customer_managed_key_deletion_grace_period"))
	return resource, data
}

type gracePeriodTestAPI struct {
	fake    *testhelpers.FakeAPI
	updates []map[string]any
}

// newGracePeriodTestAPI supplies the subscription's initial status and captures update requests.
// The fake deliberately fails updates and subsequent reads so each case stops after
// the resource's update decision and can check the resulting error.
func newGracePeriodTestAPI(status string) *gracePeriodTestAPI {
	api := &gracePeriodTestAPI{fake: testhelpers.NewFakeAPI()}
	subscriptionWasRead := false
	api.fake.Handle("GET /subscriptions/123", func(r *http.Request) (any, error) {
		if subscriptionWasRead {
			// These cases exercise request construction and error propagation, so polling ends the test.
			return nil, testhelpers.StatusError{
				Code: http.StatusForbidden,
				Body: map[string]any{"description": "subscription-polling-stopped"},
			}
		}
		subscriptionWasRead = true
		return map[string]any{"id": 123, "status": status}, nil
	})
	api.fake.Handle("PUT /subscriptions/123", func(r *http.Request) (any, error) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			return nil, err
		}
		api.updates = append(api.updates, payload)
		// Rejecting the update lets each case verify the request and the resulting error without completing activation.
		return nil, testhelpers.StatusError{
			Code: http.StatusForbidden,
			Body: map[string]any{"description": "grace-period-update-denied"},
		}
	})
	return api
}

// updateRequests returns a snapshot of the attempted subscription updates.
// Use this method to inspect captured requests without racing API handlers.
func (api *gracePeriodTestAPI) updateRequests() []map[string]any {
	var requests []map[string]any
	api.fake.WithHandlersPaused(func() {
		requests = append(requests, api.updates...)
	})
	return requests
}
