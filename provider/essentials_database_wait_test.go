package provider_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestEssentialsDatabaseActiveWaiter(t *testing.T) {
	cases := []struct {
		name      string
		states    []string
		wantError bool
	}{
		{name: "already active", states: []string{"active"}},
		{name: "redis version upgrade", states: []string{"active-upgrade-pending", "active"}},
		{name: "database update", states: []string{"bdb-update-pending", "active"}},
		{name: "change then upgrade", states: []string{"active-change-pending", "active-upgrade-pending", "active"}},
		{name: "unexpected state", states: []string{"error"}, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := testhelpers.NewFakeAPI()
			index := 0
			fake.Handle("GET /fixed/subscriptions/123/databases/456", func(r *http.Request) (any, error) {
				state := tc.states[index]
				if index < len(tc.states)-1 {
					index++
				}
				return map[string]any{"databaseId": 456, "status": state}, nil
			})
			err := provider.WaitForEssentialsDatabaseToBeActive(context.Background(), 123, 456, testhelpers.NewAPIClient(t, fake), 5*time.Second)
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			fake.WithHandlersPaused(func() {
				require.Equal(t, len(tc.states)-1, index)
			})
		})
	}
}
