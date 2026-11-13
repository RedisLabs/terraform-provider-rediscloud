package provider_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

type activeActiveTransitGatewayReadFixture struct {
	fake                *testhelpers.FakeAPI
	taskIDs             []string
	transitGatewayReads int
}

func newActiveActiveTransitGatewayReadFixture(
	apiPath string,
	apiReads [][]*attachments.TransitGatewayAttachment,
) *activeActiveTransitGatewayReadFixture {
	fixture := &activeActiveTransitGatewayReadFixture{
		fake:    testhelpers.NewFakeAPI(),
		taskIDs: make([]string, 0, len(apiReads)),
	}

	for i, tgws := range apiReads {
		taskID := fmt.Sprintf("tgw-read-%d", i+1)
		fixture.taskIDs = append(fixture.taskIDs, taskID)
		fixture.fake.Handle("GET /tasks/"+taskID, activeActiveTransitGatewayTaskHandler(taskID, tgws))
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
func activeActiveTransitGatewayTaskHandler(
	taskID string,
	tgws []*attachments.TransitGatewayAttachment,
) testhelpers.HandlerFunc {
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
					Resource: &attachments.Resource{
						TransitGatewayAttachment: tgws,
					},
				},
			}, nil
		default:
			return nil, fmt.Errorf("unexpected third read of task %s", taskID)
		}
	}
}

func (f *activeActiveTransitGatewayReadFixture) readCount() int {
	reads := 0
	f.fake.WithHandlersPaused(func() {
		reads = f.transitGatewayReads
	})
	return reads
}

func TestActiveActiveTransitGatewayResourceReadWaitsForMatchingAttachment(t *testing.T) {
	tests := []struct {
		name                    string
		resourceName            string
		givenID                 string
		givenConfig             map[string]interface{}
		givenAPIPath            string
		givenInitialAttachments []*attachments.TransitGatewayAttachment
		givenFinalAttachments   []*attachments.TransitGatewayAttachment
		wantID                  string
		wantState               map[string]interface{}
		wantAPIReads            int
	}{
		{
			name:         "attachment Read waits past an unrelated attachment",
			resourceName: "rediscloud_active_active_transit_gateway_attachment",
			givenID:      "123/456/789",
			givenConfig: map[string]interface{}{
				"subscription_id": "123",
				"region_id":       "456",
				"tgw_id":          789,
			},
			givenAPIPath: "GET /subscriptions/123/regions/456/transitGateways",
			givenInitialAttachments: []*attachments.TransitGatewayAttachment{
				{
					Id:               redis.Int(790),
					AwsTgwUid:        redis.String("tgw-790"),
					AttachmentUid:    redis.String("tgw-attach-790"),
					Status:           redis.String("available"),
					AttachmentStatus: redis.String("available"),
					AwsAccountId:     redis.String("222222222222"),
					Cidrs: []*attachments.Cidr{
						{CidrAddress: redis.String("10.0.1.0/24"), Status: redis.String("active")},
					},
				},
			},
			givenFinalAttachments: []*attachments.TransitGatewayAttachment{
				{
					Id:               redis.Int(789),
					AwsTgwUid:        redis.String("tgw-789"),
					AttachmentUid:    redis.String("tgw-attach-789"),
					Status:           redis.String("available"),
					AttachmentStatus: redis.String("available"),
					AwsAccountId:     redis.String("111111111111"),
					Cidrs: []*attachments.Cidr{
						{CidrAddress: redis.String("10.0.0.0/24"), Status: redis.String("active")},
					},
				},
			},
			wantID: "123/456/789",
			wantState: map[string]interface{}{
				"subscription_id":   "123",
				"region_id":         "456",
				"tgw_id":            789,
				"aws_tgw_uid":       "tgw-789",
				"attachment_uid":    "tgw-attach-789",
				"status":            "available",
				"attachment_status": "available",
				"aws_account_id":    "111111111111",
				"cidrs":             []interface{}{"10.0.0.0/24"},
			},
			wantAPIReads: 2,
		},
		{
			name:         "route Read waits past an unrelated attachment",
			resourceName: "rediscloud_active_active_transit_gateway_route",
			givenID:      "123/456/789",
			givenConfig: map[string]interface{}{
				"subscription_id": "123",
				"region_id":       "456",
				"tgw_id":          789,
				"cidrs":           []interface{}{"10.0.2.0/24"},
			},
			givenAPIPath: "GET /subscriptions/123/regions/456/transitGateways",
			givenInitialAttachments: []*attachments.TransitGatewayAttachment{
				{
					Id:               redis.Int(790),
					AwsTgwUid:        redis.String("tgw-790"),
					AttachmentUid:    redis.String("tgw-attach-790"),
					Status:           redis.String("available"),
					AttachmentStatus: redis.String("available"),
					AwsAccountId:     redis.String("222222222222"),
					Cidrs: []*attachments.Cidr{
						{CidrAddress: redis.String("10.0.1.0/24"), Status: redis.String("active")},
					},
				},
			},
			givenFinalAttachments: []*attachments.TransitGatewayAttachment{
				{
					Id:               redis.Int(789),
					AwsTgwUid:        redis.String("tgw-789"),
					AttachmentUid:    redis.String("tgw-attach-789"),
					Status:           redis.String("available"),
					AttachmentStatus: redis.String("available"),
					AwsAccountId:     redis.String("111111111111"),
					Cidrs: []*attachments.Cidr{
						{CidrAddress: redis.String("10.0.0.0/24"), Status: redis.String("active")},
					},
				},
			},
			wantID: "123/456/789",
			wantState: map[string]interface{}{
				"subscription_id": "123",
				"region_id":       "456",
				"tgw_id":          789,
				"cidrs":           []interface{}{"10.0.0.0/24"},
			},
			wantAPIReads: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := provider.NewSdkProvider("test")().ResourcesMap[tc.resourceName]
			data := schema.TestResourceDataRaw(t, resource.Schema, tc.givenConfig)
			data.SetId(tc.givenID)
			fixture := newActiveActiveTransitGatewayReadFixture(tc.givenAPIPath, [][]*attachments.TransitGatewayAttachment{
				tc.givenInitialAttachments,
				tc.givenFinalAttachments,
			})

			diagnostics := resource.ReadContext(context.Background(), data, testhelpers.NewAPIClient(t, fixture.fake))

			require.False(t, diagnostics.HasError(), "unexpected diagnostics: %v", diagnostics)
			require.Equal(t, tc.wantID, data.Id())
			for field, want := range tc.wantState {
				require.Equal(t, want, data.Get(field), "unexpected value for %s", field)
			}
			require.Equal(t, tc.wantAPIReads, fixture.readCount())
		})
	}
}
