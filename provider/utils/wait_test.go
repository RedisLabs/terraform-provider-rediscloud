package utils_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// The SDK exports no constants for principal types or the PrivateLink read command.
const (
	principalTypeAWSAccount   = "aws_account"
	principalTypeOrganization = "organization"
	privateLinkReadCommand    = "privateLinkGetRequest"
)

func TestWaitForPrivateLinkPrincipals(t *testing.T) {
	expected := []pl.PrivateLinkPrincipal{privateLinkPrincipal("111111111111")}

	t.Run("membership", func(t *testing.T) {
		for _, tc := range []struct {
			name        string
			success     bool
			actual      []*pl.PrivateLinkPrincipal
			wantError   string
			wantTimeout bool
		}{
			// These cases should succeed.
			{success: true, name: "associated", actual: principalWithStatus(pl.PrivateLinkPrincipalStatusAssociated)},
			{success: true, name: "invitation awaiting acceptance", actual: principalWithStatus(pl.PrivateLinkPrincipalStatusAssociating)},

			// These cases should fail.
			{success: false, name: "missing", actual: []*pl.PrivateLinkPrincipal{}, wantError: "expected [111111111111], got []", wantTimeout: true},
			{success: false, name: "extra principal", actual: apiPrincipalList("111111111111", "222222222222"), wantError: "expected [111111111111], got [111111111111 222222222222]", wantTimeout: true},
			{success: false, name: "failed", actual: principalWithStatus(pl.PrivateLinkPrincipalStatusFailed), wantError: `PrivateLink principal "111111111111" association failed`, wantTimeout: false},
			{success: false, name: "being removed", actual: principalWithStatus(pl.PrivateLinkPrincipalStatusDisassociating), wantError: "expected [111111111111], got [111111111111]", wantTimeout: true},
			{success: false, name: "removed", actual: principalWithStatus(pl.PrivateLinkPrincipalStatusDisassociated), wantError: "expected [111111111111], got [111111111111]", wantTimeout: true},
			{success: false, name: "wrong type", actual: []*pl.PrivateLinkPrincipal{{Principal: redis.String("111111111111"), Type: redis.String(principalTypeOrganization)}}, wantError: "expected [111111111111], got [111111111111]", wantTimeout: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				fake := testhelpers.NewFakeAPI()
				getPrincipals := func(*http.Request) (any, error) {
					return privateLinkReadTask(fake, tc.actual), nil
				}
				fake.Handle("GET /subscriptions/123/private-link", getPrincipals)
				api := testhelpers.NewAPIClient(t, fake)
				ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
				t.Cleanup(cancel)

				err := utils.WaitForPrivateLinkPrincipals(
					ctx, 123, api, expected, 20*time.Millisecond,
					subscriptions.SubscriptionDeploymentTypeSingleRegion, 0,
				)
				if tc.success {
					require.NoError(t, err)
					return
				}

				require.ErrorContains(t, err, tc.wantError)
				if tc.wantTimeout {
					var timeoutErr *retry.TimeoutError
					require.True(t, errors.Is(err, context.DeadlineExceeded) || errors.As(err, &timeoutErr), "%v", err)
				}
			})
		}
	})

	t.Run("Active-Active endpoint", func(t *testing.T) {
		fake := testhelpers.NewFakeAPI()
		getPrincipals := func(*http.Request) (any, error) {
			return privateLinkReadTask(fake, apiPrincipalList("111111111111")), nil
		}
		fake.Handle("GET /subscriptions/123/regions/1/private-link", getPrincipals)
		api := testhelpers.NewAPIClient(t, fake)
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		t.Cleanup(cancel)

		err := utils.WaitForPrivateLinkPrincipals(
			ctx, 123, api, expected, time.Minute,
			subscriptions.SubscriptionDeploymentTypeActiveActive, 1,
		)
		require.NoError(t, err)
	})

	t.Run("delayed membership", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name    string
			initial []*pl.PrivateLinkPrincipal
			final   []*pl.PrivateLinkPrincipal
		}{
			{
				name:    "addition",
				initial: apiPrincipalList(),
				final:   apiPrincipalList("111111111111"),
			},
			{
				name:    "deletion",
				initial: apiPrincipalList("111111111111", "222222222222"),
				final:   apiPrincipalList("111111111111"),
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				// Use synctest.Test to advance the waiter's ten-second poll interval
				// without wall-clock sleeps. See https://go.dev/blog/synctest#testing-time.
				synctest.Test(t, func(t *testing.T) {
					fake := testhelpers.NewFakeAPI()
					calls := 0
					// Model a completed write whose results are not yet visible. Return
					// stale membership once, then expose the updated membership.
					getPrincipals := func(*http.Request) (any, error) {
						calls++
						if calls == 1 {
							return privateLinkReadTask(fake, tc.initial), nil
						}
						return privateLinkReadTask(fake, tc.final), nil
					}
					fake.Handle("GET /subscriptions/123/private-link", getPrincipals)
					api := testhelpers.NewAPIClient(t, fake)
					ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
					t.Cleanup(cancel)

					err := utils.WaitForPrivateLinkPrincipals(
						ctx, 123, api, expected, 15*time.Second,
						subscriptions.SubscriptionDeploymentTypeSingleRegion, 0,
					)
					require.NoError(t, err)
					// Require a retry for stale membership and stop as soon as the final
					// membership matches the requested principals.
					assertReads := func() {
						require.Equal(t, 2, calls)
					}
					fake.WithHandlersPaused(assertReads)
				})
			})
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)
		fake := testhelpers.NewFakeAPI()
		getPrincipals := func(*http.Request) (any, error) {
			time.AfterFunc(20*time.Millisecond, cancel)
			return privateLinkReadTask(fake, apiPrincipalList()), nil
		}
		fake.Handle("GET /subscriptions/123/private-link", getPrincipals)
		api := testhelpers.NewAPIClient(t, fake)

		err := utils.WaitForPrivateLinkPrincipals(
			ctx, 123, api, expected, time.Minute,
			subscriptions.SubscriptionDeploymentTypeSingleRegion, 0,
		)
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorContains(t, err, "111111111111")
	})

	t.Run("read error", func(t *testing.T) {
		fake := testhelpers.NewFakeAPI()
		getPrincipals := func(*http.Request) (any, error) {
			return nil, testhelpers.StatusError{
				Code: http.StatusForbidden,
				Body: map[string]any{"description": "read failed"},
			}
		}
		fake.Handle("GET /subscriptions/123/private-link", getPrincipals)
		api := testhelpers.NewAPIClient(t, fake)
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		t.Cleanup(cancel)

		err := utils.WaitForPrivateLinkPrincipals(
			ctx, 123, api, expected, time.Minute,
			subscriptions.SubscriptionDeploymentTypeSingleRegion, 0,
		)
		require.ErrorContains(t, err, "verifying PrivateLink principals:")
		require.ErrorContains(t, err, "403")
		require.ErrorContains(t, err, "read failed")
	})
}

// privateLinkReadTask wraps the supplied principals in the asynchronous response
// consumed by the SDK's PrivateLink getters.
func privateLinkReadTask(fake *testhelpers.FakeAPI, principals []*pl.PrivateLinkPrincipal) map[string]any {
	link := &pl.PrivateLink{
		Status:     redis.String(pl.PrivateLinkStatusActive),
		Principals: principals,
	}
	return fake.RegisterResourceTask(privateLinkReadCommand, link)
}

func principalWithStatus(status string) []*pl.PrivateLinkPrincipal {
	principals := apiPrincipalList("111111111111")
	principals[0].Status = redis.String(status)
	return principals
}

func privateLinkPrincipal(principal string) pl.PrivateLinkPrincipal {
	return pl.PrivateLinkPrincipal{
		Principal: redis.String(principal),
		Type:      redis.String(principalTypeAWSAccount),
	}
}

func apiPrincipalList(principals ...string) []*pl.PrivateLinkPrincipal {
	result := make([]*pl.PrivateLinkPrincipal, 0, len(principals))
	for _, principal := range principals {
		value := privateLinkPrincipal(principal)
		result = append(result, &value)
	}
	return result
}
