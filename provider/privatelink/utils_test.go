package privatelink_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/privatelink"
)

// The SDK exposes principal types as strings without constants for their values.
const (
	principalTypeAWSAccount   = "aws_account"
	principalTypeOrganization = "organization"
)

func TestUnitFlattenConnections(t *testing.T) {
	// Test that FlattenConnections produces keys that match the schema
	connections := []*pl.PrivateLinkConnection{
		{
			AssociationId:   redis.String("assoc-123"),
			ConnectionId:    redis.String("conn-456"),
			Type:            redis.String("vpc-endpoint"),
			OwnerId:         redis.String("owner-789"),
			AssociationDate: redis.String("2024-01-01"),
		},
	}

	result := privatelink.FlattenConnections(connections)

	assert.Len(t, result, 1)

	conn := result[0]
	assert.Contains(t, conn, "association_id")
	assert.Contains(t, conn, "connection_id")
	assert.Contains(t, conn, "connection_type")
	assert.Contains(t, conn, "owner_id")
	assert.Contains(t, conn, "association_date")

	assert.Equal(t, "assoc-123", conn["association_id"])
	assert.Equal(t, "conn-456", conn["connection_id"])
	assert.Equal(t, "vpc-endpoint", conn["connection_type"])
	assert.Equal(t, "owner-789", conn["owner_id"])
	assert.Equal(t, "2024-01-01", conn["association_date"])
}

func TestUnitFlattenConnectionsEmpty(t *testing.T) {
	result := privatelink.FlattenConnections([]*pl.PrivateLinkConnection{})
	assert.Len(t, result, 0)
}

func TestUnitFlattenDatabases(t *testing.T) {
	databases := []*pl.PrivateLinkDatabase{
		{
			DatabaseId:           redis.Int(12345),
			Port:                 redis.Int(16379),
			ResourceLinkEndpoint: redis.String("endpoint.example.com"),
		},
	}

	result := privatelink.FlattenDatabases(databases)

	assert.Len(t, result, 1)

	db := result[0]
	assert.Contains(t, db, "database_id")
	assert.Contains(t, db, "port")
	assert.Contains(t, db, "resource_link_endpoint")

	assert.Equal(t, 12345, db["database_id"])
	assert.Equal(t, 16379, db["port"])
	assert.Equal(t, "endpoint.example.com", db["resource_link_endpoint"])
}

func TestUnitFlattenPrincipals(t *testing.T) {
	principals := []*pl.PrivateLinkPrincipal{
		{
			Principal: redis.String("arn:aws:iam::123456789012:root"),
			Type:      redis.String(principalTypeAWSAccount),
			Alias:     redis.String("my-account"),
		},
		{
			Principal: redis.String("arn:aws:iam::987654321098:root"),
			Type:      redis.String(principalTypeAWSAccount),
			Alias:     redis.String("another-account"),
		},
	}

	result := privatelink.FlattenPrincipals(principals)

	assert.Len(t, result, 2)

	// Verify keys
	for _, p := range result {
		assert.Contains(t, p, "principal")
		assert.Contains(t, p, "principal_type")
		assert.Contains(t, p, "principal_alias")
	}

	// Results should be sorted by principal
	assert.Equal(t, "arn:aws:iam::123456789012:root", result[0]["principal"])
	assert.Equal(t, "arn:aws:iam::987654321098:root", result[1]["principal"])
}

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
			{success: true, name: "associated", actual: principalWithStatus(pl.PrivateLinkPrincipalStatusAssociated)},
			{success: true, name: "invitation awaiting acceptance", actual: principalWithStatus(pl.PrivateLinkPrincipalStatusAssociating)},

			{success: false, name: "missing", actual: []*pl.PrivateLinkPrincipal{}, wantError: "expected [111111111111], got []", wantTimeout: true},
			{success: false, name: "extra principal", actual: apiPrincipalList("111111111111", "222222222222"), wantError: "expected [111111111111], got [111111111111 222222222222]", wantTimeout: true},
			{success: false, name: "failed", actual: principalWithStatus(pl.PrivateLinkPrincipalStatusFailed), wantError: `PrivateLink principal "111111111111" association failed`, wantTimeout: false},
			{success: false, name: "being removed", actual: principalWithStatus(pl.PrivateLinkPrincipalStatusDisassociating), wantError: "expected [111111111111], got [111111111111]", wantTimeout: true},
			{success: false, name: "removed", actual: principalWithStatus(pl.PrivateLinkPrincipalStatusDisassociated), wantError: "expected [111111111111], got [111111111111]", wantTimeout: true},
			{success: false, name: "wrong type", actual: []*pl.PrivateLinkPrincipal{{Principal: redis.String("111111111111"), Type: redis.String(principalTypeOrganization)}}, wantError: "expected [111111111111], got [111111111111]", wantTimeout: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				getPrincipals := func(context.Context) (*pl.PrivateLink, error) {
					return &pl.PrivateLink{Principals: tc.actual}, nil
				}
				err := privatelink.WaitForPrivateLinkPrincipals(context.Background(), getPrincipals, expected, 20*time.Millisecond)
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

	t.Run("delayed membership", func(t *testing.T) {
		t.Parallel()
		expected := []pl.PrivateLinkPrincipal{privateLinkPrincipal("111111111111")}
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
				// Use synctest.Test, so we don't  wait the actual 10 seconds on each poll
				// inside WaitForPrivateLinkPrincipals. ref: https://go.dev/blog/synctest#testing-time
				synctest.Test(t, func(t *testing.T) {
					calls := 0
					// Model a completed write whose results are not yet visible. Return
					// stale membership once, then expose the updated membership.
					getPrincipals := func(context.Context) (*pl.PrivateLink, error) {
						calls++
						if calls == 1 {
							return &pl.PrivateLink{Principals: tc.initial}, nil
						}
						return &pl.PrivateLink{Principals: tc.final}, nil
					}
					err := privatelink.WaitForPrivateLinkPrincipals(context.Background(), getPrincipals, expected, 15*time.Second)
					require.NoError(t, err)
					// Require a retry for stale membership and stop as soon as the final
					// membership matches the requested principals.
					require.Equal(t, 2, calls)
				})
			})
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		getPrincipals := func(context.Context) (*pl.PrivateLink, error) {
			time.AfterFunc(20*time.Millisecond, cancel)
			return &pl.PrivateLink{}, nil
		}
		err := privatelink.WaitForPrivateLinkPrincipals(ctx, getPrincipals, expected, time.Minute)
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorContains(t, err, "111111111111")
	})

	t.Run("read error", func(t *testing.T) {
		readErr := errors.New("read failed")
		getPrincipals := func(context.Context) (*pl.PrivateLink, error) {
			return nil, readErr
		}
		err := privatelink.WaitForPrivateLinkPrincipals(context.Background(), getPrincipals, expected, time.Minute)
		require.ErrorIs(t, err, readErr)
	})
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
