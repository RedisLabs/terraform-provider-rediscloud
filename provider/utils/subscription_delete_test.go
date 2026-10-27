package utils_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// subscriptionSequence repeats its final response until the waiter finishes.
// FakeAPI serializes calls to this handler.
func subscriptionSequence(states []string) testhelpers.HandlerFunc {
	index := 0
	return func(r *http.Request) (any, error) {
		state := states[index]
		if index < len(states)-1 {
			index++
		}
		switch state {
		case "notfound":
			return nil, testhelpers.NotFound()
		case "failure":
			return nil, testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "denied"}}
		}
		return map[string]any{"id": 123, "status": state}, nil
	}
}

func TestSubscriptionDeletionTransitions(t *testing.T) {
	cases := []struct {
		name         string
		states       []string
		deleteResult string
		deletes      int
		wantError    bool
	}{
		{name: "already absent", states: []string{"notfound"}},
		{name: "explicit deleted", states: []string{"deleted"}},
		{name: "already deleting", states: []string{"deleting", "notfound"}},
		{name: "pending to automatic deletion", states: []string{"pending", "deleting", "notfound"}},
		{name: "CMK to active", states: []string{"encryption_key_pending", "active", "active", "deleting", "notfound"}, deletes: 1},
		{name: "CMK to deleting", states: []string{"encryption_key_pending", "deleting", "notfound"}},
		{name: "CMK to absent", states: []string{"encryption_key_pending", "notfound"}},
		{name: "CMK timeout", states: []string{"encryption_key_pending"}, wantError: true},
		{name: "automatic deletion at recheck", states: []string{"active", "deleting", "notfound"}},
		{name: "absent at recheck", states: []string{"active", "notfound"}},
		{name: "CMK at recheck", states: []string{"active", "encryption_key_pending", "active", "notfound"}, deletes: 1},
		{name: "delete returns not found", states: []string{"active"}, deleteResult: "notfound", deletes: 1, wantError: true},
		{name: "delete fails", states: []string{"active"}, deleteResult: "failure", deletes: 1, wantError: true},
		{name: "polling fails after delete", states: []string{"active", "active", "failure"}, deletes: 1, wantError: true},
		{name: "deletion times out", states: []string{"active", "active", "deleting"}, deletes: 1, wantError: true},
		{name: "readiness fails", states: []string{"failure"}, wantError: true},
	}
	for _, resourceName := range []string{"rediscloud_active_active_subscription", "rediscloud_subscription"} {
		t.Run(resourceName, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					fake := testhelpers.NewFakeAPI()
					fake.Handle("GET /subscriptions/123", subscriptionSequence(tc.states))
					deletes := 0
					fake.Handle("DELETE /subscriptions/123", func(r *http.Request) (any, error) {
						deletes++
						if tc.deleteResult == "notfound" {
							return nil, testhelpers.NotFound()
						}
						if tc.deleteResult == "failure" {
							return nil, testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "denied"}}
						}
						return fake.RegisterTask("subscriptionDelete", 123), nil
					})
					api := testhelpers.NewAPIClient(t, fake)
					resource := provider.NewSdkProvider("test")().ResourcesMap[resourceName]
					require.NotNil(t, resource)
					data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{})
					data.SetId("123")
					ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
					defer cancel()
					diags := resource.DeleteContext(ctx, data, api)
					require.Equal(t, tc.wantError, diags.HasError(), "%v", diags)
					if tc.wantError {
						require.Equal(t, "123", data.Id())
					} else {
						require.Empty(t, data.Id())
					}
					fake.WithHandlersPaused(func() { require.Equal(t, tc.deletes, deletes) })
				})
			}
		})
	}
}

func TestSubscriptionStatusWaiter(t *testing.T) {
	cases := []struct {
		name                     string
		states, targets, pending []string
		want                     string
		wantError                bool
	}{
		{name: "default pending", states: []string{"pending", "active"}, targets: []string{"active"}, want: "active"},
		{name: "custom pending", states: []string{"encryption_key_pending", "active"}, targets: []string{"active"}, pending: []string{"encryption_key_pending"}, want: "active"},
		{name: "custom replaces default", states: []string{"pending"}, targets: []string{"active"}, pending: []string{"encryption_key_pending"}, wantError: true},
		{name: "returns reached target", states: []string{"deleting"}, targets: []string{"active", "deleting", "deleted"}, want: "deleting"},
		{name: "missing with deletion target", states: []string{"notfound"}, targets: []string{"active", "deleted"}, want: "deleted"},
		{name: "missing without deletion target", states: []string{"notfound"}, targets: []string{"active"}, wantError: true},
		{name: "other errors preserved", states: []string{"failure"}, targets: []string{"deleted"}, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := testhelpers.NewFakeAPI()
			fake.Handle("GET /subscriptions/123", subscriptionSequence(tc.states))
			status, err := utils.WaitForSubscriptionToBeStatus(context.Background(), 123, testhelpers.NewAPIClient(t, fake), time.Second, tc.targets, tc.pending...)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.want, status)
		})
	}
}
