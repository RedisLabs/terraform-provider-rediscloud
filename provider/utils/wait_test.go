package utils_test

import (
	"context"
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

func transitGatewayResponse(id int) map[string]any {
	return map[string]any{
		"response": map[string]any{
			"resource": map[string]any{
				"tgws": []map[string]any{{"id": id}},
			},
		},
	}
}

func TestWaitForTransitGatewayResourceToMatchFiltersContinuesPastNotFoundChecks(t *testing.T) {
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

	filters := []func(tgw *attachments.TransitGatewayAttachment) bool{
		func(tgw *attachments.TransitGatewayAttachment) bool {
			return redis.IntValue(tgw.Id) == 2
		},
	}

	task, err := utils.WaitForTransitGatewayResourceToMatchFilters(
		context.Background(),
		123,
		testhelpers.NewAPIClient(t, fake),
		filters,
		time.Second,
		subscriptions.SubscriptionDeploymentTypeSingleRegion,
		0,
	)
	require.NoError(t, err)
	require.NotNil(t, task)
	require.NotNil(t, task.Response)
	require.NotNil(t, task.Response.Resource)
	require.Len(t, task.Response.Resource.TransitGatewayAttachment, 1)
	require.Equal(t, 2, redis.IntValue(task.Response.Resource.TransitGatewayAttachment[0].Id))
	fake.WithHandlersPaused(func() {
		require.Equal(t, 43, refreshes)
	})
}
