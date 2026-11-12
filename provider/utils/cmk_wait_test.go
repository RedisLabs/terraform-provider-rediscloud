package utils_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestSubscriptionCmkWaiters(t *testing.T) {
	waiters := map[string]func(context.Context, int, *testhelpers.FakeAPI, *testing.T) error{
		"encryption key pending": func(ctx context.Context, id int, fake *testhelpers.FakeAPI, t *testing.T) error {
			return utils.WaitForSubscriptionToBeEncryptionKeyPending(ctx, id, testhelpers.NewAPIClient(t, fake), time.Second)
		},
		"active after CMK update": func(ctx context.Context, id int, fake *testhelpers.FakeAPI, t *testing.T) error {
			return utils.WaitForSubscriptionToBeActiveAfterCmkUpdate(ctx, id, testhelpers.NewAPIClient(t, fake), time.Second)
		},
	}
	cases := []struct {
		name      string
		waiter    string
		states    []string
		wantError bool
	}{
		{name: "pending to key pending", waiter: "encryption key pending", states: []string{"pending", "encryption_key_pending"}},
		{name: "pending to active", waiter: "encryption key pending", states: []string{"pending", "active"}},
		{name: "polling failure then key pending", waiter: "encryption key pending", states: []string{"failure", "encryption_key_pending"}},
		{name: "polling failure then pending", waiter: "encryption key pending", states: []string{"failure", "pending"}, wantError: true},
		{name: "error status", waiter: "encryption key pending", states: []string{"pending", "error"}, wantError: true},
		{name: "key pending to active", waiter: "active after CMK update", states: []string{"encryption_key_pending", "pending", "active"}},
		{name: "already active", waiter: "active after CMK update", states: []string{"active"}},
		{name: "stuck in key pending", waiter: "active after CMK update", states: []string{"encryption_key_pending"}, wantError: true},
		{name: "error after key pending", waiter: "active after CMK update", states: []string{"encryption_key_pending", "error"}, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.waiter+"/"+tc.name, func(t *testing.T) {
			fake := testhelpers.NewFakeAPI()
			fake.Handle("GET /subscriptions/123", subscriptionSequence(tc.states))
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := waiters[tc.waiter](ctx, 123, fake, t)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
