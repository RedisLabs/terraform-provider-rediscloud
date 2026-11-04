package pro_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func subscriptionStates(states []string) testhelpers.HandlerFunc {
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

func TestProSubscriptionDeletionTransitions(t *testing.T) {
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
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := testhelpers.NewFakeAPI()
			fake.Handle("GET /subscriptions/123", subscriptionStates(tc.states))

			deletes := 0
			fake.Handle("DELETE /subscriptions/123", func(r *http.Request) (any, error) {
				deletes++
				switch tc.deleteResult {
				case "notfound":
					return nil, testhelpers.NotFound()
				case "failure":
					return nil, testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "denied"}}
				}
				return fake.RegisterTask("subscriptionDelete", 123), nil
			})

			ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			defer cancel()

			diags := pro.DeleteProSubscription(ctx, testhelpers.NewAPIClient(t, fake), 123, 250*time.Millisecond)
			require.Equal(t, tc.wantError, diags.HasError(), "%v", diags)
			fake.WithHandlersPaused(func() {
				require.Equal(t, tc.deletes, deletes)
			})
		})
	}
}
