package transitgateway

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransitGatewayAttachmentComputedSetFromAPI(t *testing.T) {
	var model TransitGatewayAttachmentResourceModel
	diags := model.computed().setFromAPI(context.Background(), &attachments.TransitGatewayAttachment{
		AwsTgwUid:        redis.String("tgw-1"),
		AttachmentUid:    redis.String("tgw-attach-1"),
		Status:           redis.String("available"),
		AttachmentStatus: redis.String("pending-acceptance"),
		AwsAccountId:     redis.String("123456789012"),
		Cidrs:            []*attachments.Cidr{{CidrAddress: redis.String("10.0.0.0/24")}},
	})
	require.False(t, diags.HasError())

	assert.Equal(t, "tgw-1", model.AwsTgwUid.ValueString())
	assert.Equal(t, "tgw-attach-1", model.AttachmentUid.ValueString())
	assert.Equal(t, "available", model.Status.ValueString())
	assert.Equal(t, "pending-acceptance", model.AttachmentStatus.ValueString())
	assert.Equal(t, "123456789012", model.AwsAccountID.ValueString())
	assert.Equal(t, []string{"10.0.0.0/24"}, listStrings(t, model.Cidrs))
}

func TestTransitGatewayAttachmentComputedSetFromAPINilFields(t *testing.T) {
	var model ActiveActiveTransitGatewayAttachmentResourceModel
	diags := model.computed().setFromAPI(context.Background(), &attachments.TransitGatewayAttachment{})
	require.False(t, diags.HasError())

	assert.Equal(t, types.StringValue(""), model.AttachmentUid)
	assert.False(t, model.Cidrs.IsNull())
	assert.Empty(t, model.Cidrs.Elements())
}

func TestTransitGatewayAttachmentComputedSetUnknownToNull(t *testing.T) {
	model := TransitGatewayAttachmentResourceModel{
		AwsTgwUid:        types.StringUnknown(),
		AttachmentUid:    types.StringValue("tgw-attach-1"),
		Status:           types.StringUnknown(),
		AttachmentStatus: types.StringUnknown(),
		AwsAccountID:     types.StringUnknown(),
		Cidrs:            types.ListUnknown(types.StringType),
	}
	model.computed().setUnknownToNull()

	assert.True(t, model.AwsTgwUid.IsNull())
	assert.Equal(t, "tgw-attach-1", model.AttachmentUid.ValueString())
	assert.True(t, model.Status.IsNull())
	assert.True(t, model.AttachmentStatus.IsNull())
	assert.True(t, model.AwsAccountID.IsNull())
	assert.True(t, model.Cidrs.IsNull())
}

func TestTransitGatewayAttachmentComputedKeepKnownPlanValues(t *testing.T) {
	planned := TransitGatewayAttachmentResourceModel{
		AwsTgwUid:        types.StringValue("tgw-1"),
		AttachmentUid:    types.StringValue("tgw-attach-1"),
		Status:           types.StringUnknown(),
		AttachmentStatus: types.StringUnknown(),
		AwsAccountID:     types.StringValue("123456789012"),
		Cidrs:            types.ListValueMust(types.StringType, nil),
	}
	result := TransitGatewayAttachmentResourceModel{
		AwsTgwUid:        types.StringValue("tgw-1"),
		AttachmentUid:    types.StringValue("tgw-attach-2"),
		Status:           types.StringValue("available"),
		AttachmentStatus: types.StringValue("available"),
		AwsAccountID:     types.StringValue("123456789012"),
		Cidrs:            types.ListValueMust(types.StringType, nil),
	}
	result.computed().keepKnownPlanValues(planned.computed())

	assert.Equal(t, "tgw-attach-1", result.AttachmentUid.ValueString())
	assert.Equal(t, "available", result.Status.ValueString())
	assert.Equal(t, "available", result.AttachmentStatus.ValueString())
}

func TestCidrsToAPI(t *testing.T) {
	ctx := context.Background()

	for name, tc := range map[string]struct {
		cidrs types.List
		want  []string
	}{
		"null":    {cidrs: types.ListNull(types.StringType)},
		"unknown": {cidrs: types.ListUnknown(types.StringType)},
		"values": {
			cidrs: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/24"), types.StringValue("10.0.1.0/24")}),
			want:  []string{"10.0.0.0/24", "10.0.1.0/24"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, diags := cidrsToAPI(ctx, tc.cidrs)
			require.False(t, diags.HasError())
			require.NotNil(t, got)
			assert.Equal(t, tc.want, redis.StringSliceValue(got...))
		})
	}
}

func listStrings(t *testing.T, list types.List) []string {
	t.Helper()
	var values []string
	require.False(t, list.ElementsAs(context.Background(), &values, false).HasError())
	return values
}
