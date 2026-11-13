package utils_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

type transitGatewayWaitFixture struct {
	fake                *testhelpers.FakeAPI
	taskIDs             []string
	transitGatewayReads int
}

func newTransitGatewayWaitFixture(
	apiPath string,
	apiReads []*attachments.Resource,
) *transitGatewayWaitFixture {
	fixture := &transitGatewayWaitFixture{
		fake:    testhelpers.NewFakeAPI(),
		taskIDs: make([]string, 0, len(apiReads)),
	}

	for i, resource := range apiReads {
		taskID := fmt.Sprintf("tgw-wait-%d", i+1)
		fixture.taskIDs = append(fixture.taskIDs, taskID)
		fixture.fake.Handle("GET /tasks/"+taskID, transitGatewayWaitTaskHandler(taskID, resource))
	}

	fixture.fake.Handle(apiPath, func(_ *http.Request) (any, error) {
		if fixture.transitGatewayReads >= len(fixture.taskIDs) {
			return nil, fmt.Errorf("unexpected Transit Gateway read after %d scripted responses", len(fixture.taskIDs))
		}

		taskID := fixture.taskIDs[fixture.transitGatewayReads]
		fixture.transitGatewayReads++
		return map[string]any{"taskId": taskID}, nil
	})

	return fixture
}

// The SDK reads each task once to wait for completion and again to retrieve the Transit Gateway response.
func transitGatewayWaitTaskHandler(taskID string, resource *attachments.Resource) testhelpers.HandlerFunc {
	taskReads := 0

	return func(_ *http.Request) (any, error) {
		taskReads++
		switch taskReads {
		case 1:
			return &attachments.GetAttachmentsTask{
				ID:     redis.String(taskID),
				Status: redis.String("processing-completed"),
			}, nil
		case 2:
			return &attachments.GetAttachmentsTask{
				ID:     redis.String(taskID),
				Status: redis.String("processing-completed"),
				Response: &attachments.Response{
					Resource: resource,
				},
			}, nil
		default:
			return nil, fmt.Errorf("unexpected third read of task %s", taskID)
		}
	}
}

func (f *transitGatewayWaitFixture) readCount() int {
	reads := 0
	f.fake.WithHandlersPaused(func() {
		reads = f.transitGatewayReads
	})
	return reads
}

func transitGatewayResponse(id int) map[string]any {
	return map[string]any{
		"response": map[string]any{
			"resource": map[string]any{
				"tgws": []map[string]any{{"id": id}},
			},
		},
	}
}

func TestWaitForTransitGatewayAttachmentsToMatchContinuesPastNotFoundChecks(t *testing.T) {
	fake := testhelpers.NewFakeAPI()
	fake.Handle("GET /subscriptions/123/transitGateways", func(_ *http.Request) (any, error) {
		return map[string]any{"taskId": "tgw-get"}, nil
	})

	taskCalls := 0
	refreshes := 0
	fake.Handle("GET /tasks/tgw-get", func(_ *http.Request) (any, error) {
		taskCalls++
		if taskCalls%2 != 0 {
			return map[string]any{"taskId": "tgw-get", "status": "processing-completed"}, nil
		}

		refreshes++
		switch {
		case refreshes <= 21:
			return map[string]any{}, nil
		case refreshes <= 42:
			return transitGatewayResponse(1), nil
		default:
			return transitGatewayResponse(2), nil
		}
	})

	matcher := func(tgw *attachments.TransitGatewayAttachment) (bool, error) {
		return redis.IntValue(tgw.Id) == 2, nil
	}

	matches, err := utils.WaitForTransitGatewayAttachmentsToMatch(
		context.Background(),
		123,
		testhelpers.NewAPIClient(t, fake),
		matcher,
		time.Second,
		subscriptions.SubscriptionDeploymentTypeSingleRegion,
		0,
	)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Equal(t, 2, redis.IntValue(matches[0].Id))
	fake.WithHandlersPaused(func() {
		require.Equal(t, 43, refreshes)
	})
}

func TestWaitForTransitGatewayAttachmentsToMatch(t *testing.T) {
	matcherError := errors.New("attachment cannot satisfy the wait condition")

	tests := []struct {
		name             string
		subscriptionType string
		regionID         int
		givenAPIPath     string
		givenAPIReads    []*attachments.Resource
		givenMatcher     utils.TransitGatewayAttachmentMatcher
		success          bool
		wantIDs          []int
		wantError        error
		wantAPIReads     int
	}{
		// These cases should succeed.
		{
			name:             "query returns every matching attachment regardless of lifecycle state",
			subscriptionType: subscriptions.SubscriptionDeploymentTypeSingleRegion,
			regionID:         0,
			givenAPIPath:     "GET /subscriptions/123/transitGateways",
			givenAPIReads: []*attachments.Resource{
				{
					TransitGatewayAttachment: []*attachments.TransitGatewayAttachment{
						{Id: redis.Int(789), Status: redis.String("pending")},
						{Id: redis.Int(789), Status: redis.String("available")},
						{Id: redis.Int(790), Status: redis.String("available")},
					},
				},
			},
			givenMatcher: func(tgw *attachments.TransitGatewayAttachment) (bool, error) {
				return redis.IntValue(tgw.Id) == 789, nil
			},
			success:      true,
			wantIDs:      []int{789, 789},
			wantError:    nil,
			wantAPIReads: 1,
		},
		{
			name:             "Pro attachment waits through pending until available",
			subscriptionType: subscriptions.SubscriptionDeploymentTypeSingleRegion,
			regionID:         0,
			givenAPIPath:     "GET /subscriptions/123/transitGateways",
			givenAPIReads: []*attachments.Resource{
				{
					TransitGatewayAttachment: []*attachments.TransitGatewayAttachment{
						{Id: redis.Int(789), Status: redis.String("pending")},
					},
				},
				{
					TransitGatewayAttachment: []*attachments.TransitGatewayAttachment{
						{Id: redis.Int(789), Status: redis.String("available")},
					},
				},
			},
			givenMatcher: func(tgw *attachments.TransitGatewayAttachment) (bool, error) {
				if redis.IntValue(tgw.Id) != 789 {
					return false, nil
				}
				return redis.StringValue(tgw.Status) == "available", nil
			},
			success:      true,
			wantIDs:      []int{789},
			wantError:    nil,
			wantAPIReads: 2,
		},
		{
			name:             "Active-Active attachment waits through provisioning until available",
			subscriptionType: subscriptions.SubscriptionDeploymentTypeActiveActive,
			regionID:         456,
			givenAPIPath:     "GET /subscriptions/123/regions/456/transitGateways",
			givenAPIReads: []*attachments.Resource{
				nil,
				{
					TransitGatewayAttachment: []*attachments.TransitGatewayAttachment{
						{Id: redis.Int(789), Status: redis.String("available")},
					},
				},
			},
			givenMatcher: func(tgw *attachments.TransitGatewayAttachment) (bool, error) {
				if redis.IntValue(tgw.Id) != 789 {
					return false, nil
				}
				return redis.StringValue(tgw.Status) == "available", nil
			},
			success:      true,
			wantIDs:      []int{789},
			wantError:    nil,
			wantAPIReads: 2,
		},

		{
			name:             "matcher error stops polling and remains available to the caller",
			subscriptionType: subscriptions.SubscriptionDeploymentTypeSingleRegion,
			regionID:         0,
			givenAPIPath:     "GET /subscriptions/123/transitGateways",
			givenAPIReads: []*attachments.Resource{
				{
					TransitGatewayAttachment: []*attachments.TransitGatewayAttachment{
						{Id: redis.Int(789), Status: redis.String("unusable")},
					},
				},
			},
			givenMatcher: func(tgw *attachments.TransitGatewayAttachment) (bool, error) {
				if redis.IntValue(tgw.Id) == 789 {
					return false, matcherError
				}
				return false, nil
			},
			success:      false,
			wantIDs:      nil,
			wantError:    matcherError,
			wantAPIReads: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newTransitGatewayWaitFixture(tc.givenAPIPath, tc.givenAPIReads)

			matches, err := utils.WaitForTransitGatewayAttachmentsToMatch(
				context.Background(),
				123,
				testhelpers.NewAPIClient(t, fixture.fake),
				tc.givenMatcher,
				time.Second,
				tc.subscriptionType,
				tc.regionID,
			)

			require.Equal(t, tc.wantAPIReads, fixture.readCount())
			if tc.success {
				require.NoError(t, err)
				actualIDs := make([]int, 0, len(matches))
				for _, tgw := range matches {
					actualIDs = append(actualIDs, redis.IntValue(tgw.Id))
				}
				require.Equal(t, tc.wantIDs, actualIDs)
				return
			}

			require.Error(t, err)
			require.ErrorIs(t, err, tc.wantError)
			require.Nil(t, matches)
		})
	}
}
