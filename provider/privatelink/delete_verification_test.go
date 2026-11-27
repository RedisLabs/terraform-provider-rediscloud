package privatelink_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// The SDK exports no constant for the PrivateLink delete command.
const privateLinkDeleteCommand = "privateLinkDeleteRequest"

type privateLinkDeletionRead struct {
	link *pl.PrivateLink
	err  error
}

type privateLinkDeletionGiven struct {
	resourceName       string
	subscriptionID     int
	regionID           int
	id                 string
	deleteTimeout      time.Duration
	deleteError        error
	subscriptionStates []string
	subscriptionError  error
	reads              []privateLinkDeletionRead
}

// A completed task and an active parent subscription do not prove link deletion.
// Keep the Terraform ID until a PrivateLink read confirms absence. An empty
// asynchronous resource response also represents absence in the SDK.
// The SDK's PrivateLinkStatusDeleted constant means "deleting", so that status
// must keep polling until the resource is absent.
func TestPrivateLinkDeletionVerification(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		given                 privateLinkDeletionGiven
		success               bool
		wantID                string
		wantLinkReads         int
		wantSubscriptionReads int
		wantError             string
		wantTimeout           bool
	}{
		// These cases should succeed.
		{
			name: "PrivateLink/task completes before the link disappears",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_private_link",
				subscriptionID:     5101,
				regionID:           0,
				id:                 "5101",
				deleteTimeout:      30 * time.Second,
				deleteError:        nil,
				subscriptionStates: []string{subscriptions.SubscriptionStatusActive},
				subscriptionError:  nil,
				reads: []privateLinkDeletionRead{
					{link: &pl.PrivateLink{Status: redis.String(pl.PrivateLinkStatusActive)}},
					{link: &pl.PrivateLink{Status: redis.String(pl.PrivateLinkStatusDeleted)}},
					{err: testhelpers.NotFound()},
				},
			},
			success:               true,
			wantID:                "",
			wantLinkReads:         3,
			wantSubscriptionReads: 1,
			wantError:             "",
			wantTimeout:           false,
		},
		{
			name: "Active-Active PrivateLink/task completes before an empty read",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_active_active_private_link",
				subscriptionID:     5102,
				regionID:           1,
				id:                 "5102/1",
				deleteTimeout:      30 * time.Second,
				deleteError:        nil,
				subscriptionStates: []string{subscriptions.SubscriptionStatusActive},
				subscriptionError:  nil,
				reads: []privateLinkDeletionRead{
					{link: &pl.PrivateLink{Status: redis.String(pl.PrivateLinkStatusActive)}},
					{link: &pl.PrivateLink{Status: redis.String(pl.PrivateLinkStatusDeleted)}},
					{link: &pl.PrivateLink{}},
				},
			},
			success:               true,
			wantID:                "",
			wantLinkReads:         3,
			wantSubscriptionReads: 1,
			wantError:             "",
			wantTimeout:           false,
		},
		{
			name: "PrivateLink/delete reports already absent",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_private_link",
				subscriptionID:     5103,
				regionID:           0,
				id:                 "5103",
				deleteTimeout:      30 * time.Second,
				deleteError:        testhelpers.NotFound(),
				subscriptionStates: []string{subscriptions.SubscriptionStatusActive},
				subscriptionError:  nil,
				reads:              []privateLinkDeletionRead{},
			},
			success:               true,
			wantID:                "",
			wantLinkReads:         0,
			wantSubscriptionReads: 0,
			wantError:             "",
			wantTimeout:           false,
		},
		{
			name: "Active-Active PrivateLink/delete reports already absent",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_active_active_private_link",
				subscriptionID:     5104,
				regionID:           1,
				id:                 "5104/1",
				deleteTimeout:      30 * time.Second,
				deleteError:        testhelpers.NotFound(),
				subscriptionStates: []string{subscriptions.SubscriptionStatusActive},
				subscriptionError:  nil,
				reads:              []privateLinkDeletionRead{},
			},
			success:               true,
			wantID:                "",
			wantLinkReads:         0,
			wantSubscriptionReads: 0,
			wantError:             "",
			wantTimeout:           false,
		},

		// These cases should fail.
		{
			name: "PrivateLink/completed task leaves the link active",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_private_link",
				subscriptionID:     5201,
				regionID:           0,
				id:                 "5201",
				deleteTimeout:      15 * time.Second,
				deleteError:        nil,
				subscriptionStates: []string{subscriptions.SubscriptionStatusActive},
				subscriptionError:  nil,
				reads: []privateLinkDeletionRead{
					{link: &pl.PrivateLink{Status: redis.String(pl.PrivateLinkStatusActive)}},
				},
			},
			success:               false,
			wantID:                "5201",
			wantLinkReads:         2,
			wantSubscriptionReads: 1,
			wantError:             "verifying PrivateLink deletion",
			wantTimeout:           true,
		},
		{
			name: "Active-Active PrivateLink/completed task leaves the link deleting",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_active_active_private_link",
				subscriptionID:     5202,
				regionID:           1,
				id:                 "5202/1",
				deleteTimeout:      15 * time.Second,
				deleteError:        nil,
				subscriptionStates: []string{subscriptions.SubscriptionStatusActive},
				subscriptionError:  nil,
				reads: []privateLinkDeletionRead{
					{link: &pl.PrivateLink{Status: redis.String(pl.PrivateLinkStatusDeleted)}},
				},
			},
			success:               false,
			wantID:                "5202/1",
			wantLinkReads:         2,
			wantSubscriptionReads: 1,
			wantError:             "verifying PrivateLink deletion",
			wantTimeout:           true,
		},
		{
			name: "PrivateLink/verification read fails",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_private_link",
				subscriptionID:     5203,
				regionID:           0,
				id:                 "5203",
				deleteTimeout:      30 * time.Second,
				deleteError:        nil,
				subscriptionStates: []string{subscriptions.SubscriptionStatusActive},
				subscriptionError:  nil,
				reads: []privateLinkDeletionRead{
					{err: testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "PrivateLink read denied"}}},
				},
			},
			success:               false,
			wantID:                "5203",
			wantLinkReads:         1,
			wantSubscriptionReads: 1,
			wantError:             "PrivateLink read denied",
			wantTimeout:           false,
		},
		{
			name: "Active-Active PrivateLink/verification read fails",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_active_active_private_link",
				subscriptionID:     5204,
				regionID:           1,
				id:                 "5204/1",
				deleteTimeout:      30 * time.Second,
				deleteError:        nil,
				subscriptionStates: []string{subscriptions.SubscriptionStatusActive},
				subscriptionError:  nil,
				reads: []privateLinkDeletionRead{
					{err: testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "PrivateLink read denied"}}},
				},
			},
			success:               false,
			wantID:                "5204/1",
			wantLinkReads:         1,
			wantSubscriptionReads: 1,
			wantError:             "PrivateLink read denied",
			wantTimeout:           false,
		},
		{
			name: "PrivateLink/delete fails",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_private_link",
				subscriptionID:     5205,
				regionID:           0,
				id:                 "5205",
				deleteTimeout:      30 * time.Second,
				deleteError:        testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "delete denied"}},
				subscriptionStates: []string{subscriptions.SubscriptionStatusActive},
				subscriptionError:  nil,
				reads:              []privateLinkDeletionRead{},
			},
			success:               false,
			wantID:                "5205",
			wantLinkReads:         0,
			wantSubscriptionReads: 0,
			wantError:             "delete denied",
			wantTimeout:           false,
		},
		{
			name: "Active-Active PrivateLink/delete fails",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_active_active_private_link",
				subscriptionID:     5206,
				regionID:           1,
				id:                 "5206/1",
				deleteTimeout:      30 * time.Second,
				deleteError:        testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "delete denied"}},
				subscriptionStates: []string{subscriptions.SubscriptionStatusActive},
				subscriptionError:  nil,
				reads:              []privateLinkDeletionRead{},
			},
			success:               false,
			wantID:                "5206/1",
			wantLinkReads:         0,
			wantSubscriptionReads: 0,
			wantError:             "delete denied",
			wantTimeout:           false,
		},
		{
			name: "PrivateLink/subscription readiness read fails",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_private_link",
				subscriptionID:     5207,
				regionID:           0,
				id:                 "5207",
				deleteTimeout:      30 * time.Second,
				deleteError:        nil,
				subscriptionStates: []string{subscriptions.SubscriptionStatusActive},
				subscriptionError:  testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "subscription read denied"}},
				reads:              []privateLinkDeletionRead{},
			},
			success:               false,
			wantID:                "5207",
			wantLinkReads:         0,
			wantSubscriptionReads: 1,
			wantError:             "subscription read denied",
			wantTimeout:           false,
		},
		{
			name: "Active-Active PrivateLink/subscription readiness read fails",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_active_active_private_link",
				subscriptionID:     5208,
				regionID:           1,
				id:                 "5208/1",
				deleteTimeout:      30 * time.Second,
				deleteError:        nil,
				subscriptionStates: []string{subscriptions.SubscriptionStatusActive},
				subscriptionError:  testhelpers.StatusError{Code: http.StatusForbidden, Body: map[string]any{"description": "subscription read denied"}},
				reads:              []privateLinkDeletionRead{},
			},
			success:               false,
			wantID:                "5208/1",
			wantLinkReads:         0,
			wantSubscriptionReads: 1,
			wantError:             "subscription read denied",
			wantTimeout:           false,
		},
		{
			name: "PrivateLink/readiness and verification share the delete timeout",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_private_link",
				subscriptionID:     5209,
				regionID:           0,
				id:                 "5209",
				deleteTimeout:      15 * time.Second,
				deleteError:        nil,
				subscriptionStates: []string{subscriptions.SubscriptionStatusPending, subscriptions.SubscriptionStatusActive},
				subscriptionError:  nil,
				reads: []privateLinkDeletionRead{
					{link: &pl.PrivateLink{Status: redis.String(pl.PrivateLinkStatusActive)}},
					{err: testhelpers.NotFound()},
				},
			},
			success:               false,
			wantID:                "5209",
			wantLinkReads:         1,
			wantSubscriptionReads: 2,
			wantError:             "context deadline exceeded",
			wantTimeout:           true,
		},
		{
			name: "Active-Active PrivateLink/readiness and verification share the delete timeout",
			given: privateLinkDeletionGiven{
				resourceName:       "rediscloud_active_active_private_link",
				subscriptionID:     5210,
				regionID:           1,
				id:                 "5210/1",
				deleteTimeout:      15 * time.Second,
				deleteError:        nil,
				subscriptionStates: []string{subscriptions.SubscriptionStatusPending, subscriptions.SubscriptionStatusActive},
				subscriptionError:  nil,
				reads: []privateLinkDeletionRead{
					{link: &pl.PrivateLink{Status: redis.String(pl.PrivateLinkStatusActive)}},
					{link: &pl.PrivateLink{}},
				},
			},
			success:               false,
			wantID:                "5210/1",
			wantLinkReads:         1,
			wantSubscriptionReads: 2,
			wantError:             "context deadline exceeded",
			wantTimeout:           true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Advance polling intervals and operation deadlines with fake time.
			// See https://go.dev/blog/synctest#testing-time.
			synctest.Test(t, func(t *testing.T) {
				resource, data := newPrivateLinkDeletionData(t, tc.given)
				fixture := newPrivateLinkDeletionAPI(tc.given)
				api := testhelpers.NewAPIClient(t, fixture.fake)
				api.WaitPollIntervalOverride = 10 * time.Second
				// Leave the parent deadline longer than the configured Delete timeout
				// so the handler must enforce its own operation budget.
				ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
				t.Cleanup(cancel)
				started := time.Now()

				diagnostics := resource.DeleteContext(ctx, data, api)

				deletes, linkReads, subscriptionReads := fixture.requestCounts()
				require.Equal(t, 1, deletes)
				require.Equal(t, tc.wantLinkReads, linkReads)
				require.Equal(t, tc.wantSubscriptionReads, subscriptionReads)
				require.Equal(t, tc.wantID, data.Id())
				require.LessOrEqual(t, time.Since(started), tc.given.deleteTimeout)
				if tc.success {
					require.False(t, diagnostics.HasError(), "%v", diagnostics)
					return
				}

				require.True(t, diagnostics.HasError(), "%v", diagnostics)
				require.Contains(t, fmt.Sprint(diagnostics), tc.wantError)
				if tc.wantTimeout {
					require.Regexp(t, "context deadline exceeded|timeout while waiting", fmt.Sprint(diagnostics))
				}
			})
		})
	}
}

// newPrivateLinkDeletionData attaches the case's timeout to the ResourceData
// because direct handler calls bypass the SDK's Apply setup.
func newPrivateLinkDeletionData(t *testing.T, given privateLinkDeletionGiven) (*schema.Resource, *schema.ResourceData) {
	t.Helper()
	resource := provider.NewSdkProvider("test")().ResourcesMap[given.resourceName]
	require.NotNil(t, resource)
	resource.Timeouts.Delete = schema.DefaultTimeout(given.deleteTimeout)
	config := map[string]interface{}{
		"subscription_id": fmt.Sprint(given.subscriptionID),
		"share_name":      "test-share",
		"principal":       principalTestConfig([]string{"111111111111"}),
	}
	if given.regionID != 0 {
		config["region_id"] = given.regionID
	}
	data := schema.TestResourceDataRaw(t, resource.Schema, config)
	data.SetId(given.id)
	data = resource.Data(data.State())
	require.Equal(t, given.deleteTimeout, data.Timeout(schema.TimeoutDelete))
	return resource, data
}

type privateLinkDeletionAPI struct {
	fake               *testhelpers.FakeAPI
	subscriptionID     int
	deleteError        error
	reads              []privateLinkDeletionRead
	subscriptionStates []string
	subscriptionError  error
	deletes            int
	linkReads          int
	subscriptionReads  int
}

// newPrivateLinkDeletionAPI keeps task completion independent from subsequent
// PrivateLink reads. The subscription remains present throughout deletion.
func newPrivateLinkDeletionAPI(given privateLinkDeletionGiven) *privateLinkDeletionAPI {
	fixture := &privateLinkDeletionAPI{
		fake:               testhelpers.NewFakeAPI(),
		subscriptionID:     given.subscriptionID,
		deleteError:        given.deleteError,
		reads:              given.reads,
		subscriptionStates: given.subscriptionStates,
		subscriptionError:  given.subscriptionError,
	}
	path := fmt.Sprintf("/subscriptions/%d/private-link", given.subscriptionID)
	if given.regionID != 0 {
		path = fmt.Sprintf("/subscriptions/%d/regions/%d/private-link", given.subscriptionID, given.regionID)
	}
	fixture.fake.Handle("DELETE "+path, fixture.deletePrivateLink)
	fixture.fake.Handle("GET "+path, fixture.readPrivateLink)
	fixture.fake.Handle("GET /subscriptions/"+fmt.Sprint(given.subscriptionID), fixture.readSubscription)
	return fixture
}

func (fixture *privateLinkDeletionAPI) deletePrivateLink(*http.Request) (any, error) {
	fixture.deletes++
	if fixture.deleteError != nil {
		return nil, fixture.deleteError
	}
	return fixture.fake.RegisterTask(privateLinkDeleteCommand, fixture.subscriptionID), nil
}

// readPrivateLink returns the scripted results in order and repeats the final
// result so a link that never disappears remains visible until timeout.
func (fixture *privateLinkDeletionAPI) readPrivateLink(*http.Request) (any, error) {
	fixture.linkReads++
	if len(fixture.reads) == 0 {
		return nil, fmt.Errorf("no PrivateLink deletion read configured")
	}
	result := fixture.reads[0]
	if len(fixture.reads) > 1 {
		fixture.reads = fixture.reads[1:]
	}
	if result.err != nil {
		return nil, result.err
	}
	return fixture.fake.RegisterResourceTask(privateLinkReadCommand, result.link), nil
}

// readSubscription repeats its final status to keep parent readiness independent
// from whether the PrivateLink disappears.
func (fixture *privateLinkDeletionAPI) readSubscription(*http.Request) (any, error) {
	fixture.subscriptionReads++
	if fixture.subscriptionError != nil {
		return nil, fixture.subscriptionError
	}
	if len(fixture.subscriptionStates) == 0 {
		return nil, fmt.Errorf("no subscription status configured")
	}
	status := fixture.subscriptionStates[0]
	if len(fixture.subscriptionStates) > 1 {
		fixture.subscriptionStates = fixture.subscriptionStates[1:]
	}
	return map[string]any{"id": fixture.subscriptionID, "status": status}, nil
}

// requestCounts snapshots fixture state while API handlers are paused.
func (fixture *privateLinkDeletionAPI) requestCounts() (deletes, linkReads, subscriptionReads int) {
	readCounts := func() {
		deletes = fixture.deletes
		linkReads = fixture.linkReads
		subscriptionReads = fixture.subscriptionReads
	}
	fixture.fake.WithHandlersPaused(readCounts)
	return
}
