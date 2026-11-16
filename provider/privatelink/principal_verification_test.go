package privatelink_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// These command labels are fixture metadata. The SDK exports no constants for them.
const (
	privateLinkReadCommand            = "privateLinkGetRequest"
	privateLinkCreateCommand          = "privateLinkCreateRequest"
	privateLinkCreatePrincipalCommand = "privateLinkCreatePrincipalRequest"
	privateLinkDeletePrincipalCommand = "privateLinkDeletePrincipalRequest"
)

// Describe the configured account principals together with the membership returned
// by the initial link read and by subsequent verification reads.
type principalVerificationLink struct {
	resourceName   string
	subscriptionID int
	regionID       int
	requested      []string
	initial        []*pl.PrivateLinkPrincipal
	final          []*pl.PrivateLinkPrincipal
}

// A successful ceate task does not prove that the requested principals exist.
// Create must verify membership and retain the resource ID and configured
// principals when verification fails.
func TestPrivateLinkPrincipalVerificationCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		given       principalVerificationLink
		success     bool
		wantCreates int
		wantDeletes int
		wantID      string
		wantError   string
	}{
		// These cases should succeed.
		{
			name: "PrivateLink/all requested principals appear",
			given: principalVerificationLink{
				resourceName:   "rediscloud_private_link",
				subscriptionID: 1001,
				regionID:       0,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("111111111111"),
				final:          apiPrincipalList("111111111111", "222222222222"),
			},
			success:     true,
			wantCreates: 2,
			wantDeletes: 0,
			wantID:      "1001",
			wantError:   "",
		},
		{
			name: "Active-Active PrivateLink/all requested principals appear",
			given: principalVerificationLink{
				resourceName:   "rediscloud_active_active_private_link",
				subscriptionID: 3001,
				regionID:       1,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("111111111111"),
				final:          apiPrincipalList("111111111111", "222222222222"),
			},
			success:     true,
			wantCreates: 2,
			wantDeletes: 0,
			wantID:      "3001/1",
			wantError:   "",
		},

		// These cases should return an error.
		{
			name: "PrivateLink/first requested principal is missing",
			given: principalVerificationLink{
				resourceName:   "rediscloud_private_link",
				subscriptionID: 1002,
				regionID:       0,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList(),
				final:          apiPrincipalList("222222222222"),
			},
			success:     false,
			wantCreates: 2,
			wantDeletes: 0,
			wantID:      "1002",
			wantError:   "expected [111111111111 222222222222], got [222222222222]",
		},
		{
			name: "PrivateLink/additional requested principal is missing",
			given: principalVerificationLink{
				resourceName:   "rediscloud_private_link",
				subscriptionID: 1003,
				regionID:       0,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("111111111111"),
				final:          apiPrincipalList("111111111111"),
			},
			success:     false,
			wantCreates: 2,
			wantDeletes: 0,
			wantID:      "1003",
			wantError:   "expected [111111111111 222222222222], got [111111111111]",
		},
		{
			name: "PrivateLink/an extra principal is present",
			given: principalVerificationLink{
				resourceName:   "rediscloud_private_link",
				subscriptionID: 1004,
				regionID:       0,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("111111111111"),
				final:          apiPrincipalList("111111111111", "222222222222", "333333333333"),
			},
			success:     false,
			wantCreates: 2,
			wantDeletes: 0,
			wantID:      "1004",
			wantError:   "expected [111111111111 222222222222], got [111111111111 222222222222 333333333333]",
		},
		{
			name: "Active-Active PrivateLink/first requested principal is missing",
			given: principalVerificationLink{
				resourceName:   "rediscloud_active_active_private_link",
				subscriptionID: 3002,
				regionID:       1,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList(),
				final:          apiPrincipalList("222222222222"),
			},
			success:     false,
			wantCreates: 2,
			wantDeletes: 0,
			wantID:      "3002/1",
			wantError:   "expected [111111111111 222222222222], got [222222222222]",
		},
		{
			name: "Active-Active PrivateLink/additional requested principal is missing",
			given: principalVerificationLink{
				resourceName:   "rediscloud_active_active_private_link",
				subscriptionID: 3003,
				regionID:       1,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("111111111111"),
				final:          apiPrincipalList("111111111111"),
			},
			success:     false,
			wantCreates: 2,
			wantDeletes: 0,
			wantID:      "3003/1",
			wantError:   "expected [111111111111 222222222222], got [111111111111]",
		},
		{
			name: "Active-Active PrivateLink/an extra principal is present",
			given: principalVerificationLink{
				resourceName:   "rediscloud_active_active_private_link",
				subscriptionID: 3004,
				regionID:       1,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("111111111111"),
				final:          apiPrincipalList("111111111111", "222222222222", "333333333333"),
			},
			success:     false,
			wantCreates: 2,
			wantDeletes: 0,
			wantID:      "3004/1",
			wantError:   "expected [111111111111 222222222222], got [111111111111 222222222222 333333333333]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource, data := newPrincipalVerificationResourceData(t, tc.given)
			ctx, fixture := newPrincipalVerificationAPI(t, tc.given, tc.success)

			diagnostics := resource.CreateContext(ctx, data, testhelpers.NewAPIClient(t, fixture.fake))

			creates, deletes := fixture.mutationCounts()
			require.Equal(t, tc.wantCreates, creates)
			require.Equal(t, tc.wantDeletes, deletes)
			// Retain the ID and requested principals for recovery when verification
			// fails before the final state refresh.
			require.Equal(t, tc.wantID, data.Id())
			require.ElementsMatch(t, principalTestConfig(tc.given.requested), data.Get("principal").(*schema.Set).List())

			if tc.success {
				require.False(t, diagnostics.HasError(), "%v", diagnostics)
				return
			}

			require.True(t, diagnostics.HasError(), "%v", diagnostics)
			require.Contains(t, fmt.Sprint(diagnostics), "verifying PrivateLink principals")
			require.Contains(t, fmt.Sprint(diagnostics), tc.wantError)
		})
	}
}

// Successful addition and deletion tasks can leave incomplete membership.
// Update must verify the final set before refreshing Terraform state.
func TestPrivateLinkPrincipalVerificationUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		given       principalVerificationLink
		success     bool
		wantCreates int
		wantDeletes int
		wantID      string
		wantError   string
	}{
		// These cases should succeed.
		{
			name: "PrivateLink/all requested principals appear",
			given: principalVerificationLink{
				resourceName:   "rediscloud_private_link",
				subscriptionID: 2001,
				regionID:       0,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("333333333333"),
				final:          apiPrincipalList("111111111111", "222222222222"),
			},
			success:     true,
			wantCreates: 2,
			wantDeletes: 1,
			wantID:      "2001",
			wantError:   "",
		},
		{
			name: "Active-Active PrivateLink/all requested principals appear",
			given: principalVerificationLink{
				resourceName:   "rediscloud_active_active_private_link",
				subscriptionID: 4001,
				regionID:       1,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("333333333333"),
				final:          apiPrincipalList("111111111111", "222222222222"),
			},
			success:     true,
			wantCreates: 2,
			wantDeletes: 1,
			wantID:      "4001/1",
			wantError:   "",
		},

		// These cases should return an error.
		{
			name: "PrivateLink/first requested principal is missing",
			given: principalVerificationLink{
				resourceName:   "rediscloud_private_link",
				subscriptionID: 2002,
				regionID:       0,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("333333333333"),
				final:          apiPrincipalList("222222222222"),
			},
			success:     false,
			wantCreates: 2,
			wantDeletes: 1,
			wantID:      "2002",
			wantError:   "expected [111111111111 222222222222], got [222222222222]",
		},
		{
			name: "PrivateLink/additional requested principal is missing",
			given: principalVerificationLink{
				resourceName:   "rediscloud_private_link",
				subscriptionID: 2003,
				regionID:       0,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("333333333333"),
				final:          apiPrincipalList("111111111111"),
			},
			success:     false,
			wantCreates: 2,
			wantDeletes: 1,
			wantID:      "2003",
			wantError:   "expected [111111111111 222222222222], got [111111111111]",
		},
		{
			name: "PrivateLink/an extra principal is present",
			given: principalVerificationLink{
				resourceName:   "rediscloud_private_link",
				subscriptionID: 2004,
				regionID:       0,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("333333333333"),
				final:          apiPrincipalList("111111111111", "222222222222", "333333333333"),
			},
			success:     false,
			wantCreates: 2,
			wantDeletes: 1,
			wantID:      "2004",
			wantError:   "expected [111111111111 222222222222], got [111111111111 222222222222 333333333333]",
		},
		{
			name: "Active-Active PrivateLink/first requested principal is missing",
			given: principalVerificationLink{
				resourceName:   "rediscloud_active_active_private_link",
				subscriptionID: 4002,
				regionID:       1,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("333333333333"),
				final:          apiPrincipalList("222222222222"),
			},
			success:     false,
			wantCreates: 2,
			wantDeletes: 1,
			wantID:      "4002/1",
			wantError:   "expected [111111111111 222222222222], got [222222222222]",
		},
		{
			name: "Active-Active PrivateLink/additional requested principal is missing",
			given: principalVerificationLink{
				resourceName:   "rediscloud_active_active_private_link",
				subscriptionID: 4003,
				regionID:       1,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("333333333333"),
				final:          apiPrincipalList("111111111111"),
			},
			success:     false,
			wantCreates: 2,
			wantDeletes: 1,
			wantID:      "4003/1",
			wantError:   "expected [111111111111 222222222222], got [111111111111]",
		},
		{
			name: "Active-Active PrivateLink/an extra principal is present",
			given: principalVerificationLink{
				resourceName:   "rediscloud_active_active_private_link",
				subscriptionID: 4004,
				regionID:       1,
				requested:      []string{"111111111111", "222222222222"},
				initial:        apiPrincipalList("333333333333"),
				final:          apiPrincipalList("111111111111", "222222222222", "333333333333"),
			},
			success:     false,
			wantCreates: 2,
			wantDeletes: 1,
			wantID:      "4004/1",
			wantError:   "expected [111111111111 222222222222], got [111111111111 222222222222 333333333333]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource, data := newPrincipalVerificationUpdateData(t, tc.given)
			ctx, fixture := newPrincipalVerificationAPI(t, tc.given, tc.success)

			diagnostics := resource.UpdateContext(ctx, data, testhelpers.NewAPIClient(t, fixture.fake))

			creates, deletes := fixture.mutationCounts()
			require.Equal(t, tc.wantCreates, creates)
			require.Equal(t, tc.wantDeletes, deletes)
			// Retain the ID and requested principals for recovery when verification
			// fails before the final state refresh.
			require.Equal(t, tc.wantID, data.Id())
			require.ElementsMatch(t, principalTestConfig(tc.given.requested), data.Get("principal").(*schema.Set).List())

			if tc.success {
				require.False(t, diagnostics.HasError(), "%v", diagnostics)
				return
			}

			require.True(t, diagnostics.HasError(), "%v", diagnostics)
			require.Contains(t, fmt.Sprint(diagnostics), "verifying PrivateLink principals")
			require.Contains(t, fmt.Sprint(diagnostics), tc.wantError)
		})
	}
}

// newPrincipalVerificationResourceData prepares the configured account principals
// for a direct handler call. The returned resource data has no Terraform ID.
func newPrincipalVerificationResourceData(t *testing.T, given principalVerificationLink) (*schema.Resource, *schema.ResourceData) {
	t.Helper()

	resource := provider.NewSdkProvider("test")().ResourcesMap[given.resourceName]
	require.NotNil(t, resource, "unknown resource %q", given.resourceName)
	config := map[string]interface{}{
		"subscription_id": fmt.Sprint(given.subscriptionID),
		"share_name":      "test-share",
		"principal":       principalTestConfig(given.requested),
	}
	if given.regionID != 0 {
		config["region_id"] = given.regionID
	}
	return resource, schema.TestResourceDataRaw(t, resource.Schema, config)
}

// newPrincipalVerificationUpdateData marks the link as an existing resource while
// retaining the planned principal change, so Update enters its mutation path.
func newPrincipalVerificationUpdateData(t *testing.T, given principalVerificationLink) (*schema.Resource, *schema.ResourceData) {
	t.Helper()

	resource, data := newPrincipalVerificationResourceData(t, given)
	id := fmt.Sprint(given.subscriptionID)
	if given.regionID != 0 {
		id = fmt.Sprintf("%d/%d", given.subscriptionID, given.regionID)
	}
	data.SetId(id)
	require.True(t, data.HasChange("principal"))
	return resource, data
}

func principalTestConfig(principals []string) []interface{} {
	result := make([]interface{}, 0, len(principals))
	for _, principal := range principals {
		result = append(result, map[string]interface{}{
			"principal":       principal,
			"principal_type":  principalTypeAWSAccount,
			"principal_alias": "",
		})
	}
	return result
}

type principalVerificationAPI struct {
	fake                 *testhelpers.FakeAPI
	subscriptionID       int
	privateLinkPath      string
	principalReads       [][]*pl.PrivateLinkPrincipal
	cancelAfterFinalRead context.CancelFunc
	creates              int
	deletes              int
}

// newPrincipalVerificationAPI supplies successful tasks with independently
// configured principal lists. The first read returns given.initial for link
// readiness or update diffing. Later reads repeat given.final for verification
// and state refresh. Give parallel cases distinct subscription IDs because
// resource handlers lock subscriptions.
//
// Pass success=false to cancel shortly after the final membership response.
// The cancellation bounds mismatch cases while allowing diagnostics to include
// the observed list. These cases exercise error propagation and state
// preservation on cancellation.
func newPrincipalVerificationAPI(t *testing.T, given principalVerificationLink, success bool) (context.Context, *principalVerificationAPI) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Second)
	t.Cleanup(cancel)
	path := fmt.Sprintf("/subscriptions/%d/private-link", given.subscriptionID)
	if given.regionID != 0 {
		// Handle active active
		path = fmt.Sprintf("/subscriptions/%d/regions/%d/private-link", given.subscriptionID, given.regionID)
	}
	fixture := &principalVerificationAPI{
		fake:            testhelpers.NewFakeAPI(),
		subscriptionID:  given.subscriptionID,
		privateLinkPath: path,
		principalReads:  [][]*pl.PrivateLinkPrincipal{given.initial, given.final},
	}
	if !success {
		fixture.cancelAfterFinalRead = cancel
	}
	fixture.registerHandlers()
	return ctx, fixture
}

// registerHandlers serves the link and its subscription through the shared fake.
// Create requests and principal deletions are recorded for membership assertions.
func (fixture *principalVerificationAPI) registerHandlers() {
	fixture.fake.Handle("GET /subscriptions/"+fmt.Sprint(fixture.subscriptionID), fixture.readSubscription)
	fixture.fake.Handle("GET "+fixture.privateLinkPath, fixture.readPrivateLink)
	fixture.fake.Handle("POST "+fixture.privateLinkPath, fixture.createPrivateLink)
	fixture.fake.Handle("POST "+fixture.privateLinkPath+"/principals", fixture.createPrincipal)
	fixture.fake.Handle("DELETE "+fixture.privateLinkPath+"/principals", fixture.deletePrincipal)
}

func (fixture *principalVerificationAPI) readSubscription(*http.Request) (any, error) {
	// An active subscription lets Create reach principal verification.
	return map[string]any{"id": fixture.subscriptionID, "status": subscriptions.SubscriptionStatusActive}, nil
}

// readPrivateLink serves the principal lists in fixture.principalReads in order.
// The final list repeats so verification and state refresh see the same membership.
func (fixture *principalVerificationAPI) readPrivateLink(*http.Request) (any, error) {
	if len(fixture.principalReads) == 0 {
		return nil, fmt.Errorf("no PrivateLink read response configured")
	}
	principals := fixture.principalReads[0]
	// Verification and the final state refresh both need the last response.
	// Keep that entry available for repeated reads. Successful cases leave the
	// cancellation callback nil, and mismatch cases clear it after scheduling it once.
	if len(fixture.principalReads) > 1 {
		fixture.principalReads = fixture.principalReads[1:]
	} else if fixture.cancelAfterFinalRead != nil {
		time.AfterFunc(20*time.Millisecond, fixture.cancelAfterFinalRead)
		fixture.cancelAfterFinalRead = nil
	}
	link := &pl.PrivateLink{
		ShareName:  redis.String("test-share"),
		Status:     redis.String(pl.PrivateLinkStatusActive),
		Principals: principals,
	}
	return fixture.fake.RegisterResourceTask(privateLinkReadCommand, link), nil
}

func (fixture *principalVerificationAPI) createPrivateLink(*http.Request) (any, error) {
	// Creating a PrivateLink also requests its first principal, so count this
	// request as a principal creation attempt.
	fixture.creates++
	return fixture.fake.RegisterTask(privateLinkCreateCommand, fixture.subscriptionID), nil
}

func (fixture *principalVerificationAPI) createPrincipal(*http.Request) (any, error) {
	fixture.creates++
	return fixture.fake.RegisterTask(privateLinkCreatePrincipalCommand, fixture.subscriptionID), nil
}

func (fixture *principalVerificationAPI) deletePrincipal(*http.Request) (any, error) {
	fixture.deletes++
	return fixture.fake.RegisterTask(privateLinkDeletePrincipalCommand, fixture.subscriptionID), nil
}

// mutationCounts returns a consistent snapshot of attempted principal creations and
// deletions while API handlers are paused.
func (fixture *principalVerificationAPI) mutationCounts() (creates, deletes int) {
	readCounts := func() {
		creates = fixture.creates
		deletes = fixture.deletes
	}
	fixture.fake.WithHandlersPaused(readCounts)
	return
}
