package transitgateway_test

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/transitgateway"
)

func TestTransitGatewayInvitationsFromAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("nil response is a known empty list", func(t *testing.T) {
		list, diags := transitgateway.TransitGatewayInvitationsFromAPI(ctx, nil)
		require.False(t, diags.HasError())
		assert.False(t, list.IsNull())
		assert.False(t, list.IsUnknown())
		assert.Empty(t, list.Elements())
	})

	t.Run("maps fields, keeps empty strings and sorts by id", func(t *testing.T) {
		list, diags := transitgateway.TransitGatewayInvitationsFromAPI(ctx, []*attachments.TransitGatewayInvitation{
			{Id: redis.Int(20), Name: redis.String("second"), Status: redis.String("pending")},
			nil,
			{
				Id:               redis.Int(10),
				Name:             redis.String("first"),
				ResourceShareUid: redis.String("arn:aws:ram:share"),
				AwsAccountId:     redis.String("123456789012"),
				Status:           redis.String("accepted"),
				SharedDate:       redis.String("2026-01-01T00:00:00Z"),
			},
		})
		require.False(t, diags.HasError())

		var models []transitgateway.TransitGatewayInvitationModel
		require.False(t, list.ElementsAs(ctx, &models, false).HasError())
		require.Len(t, models, 2)

		assert.Equal(t, int64(10), models[0].ID.ValueInt64())
		assert.Equal(t, "first", models[0].Name.ValueString())
		assert.Equal(t, "arn:aws:ram:share", models[0].ResourceShareUID.ValueString())
		assert.Equal(t, "123456789012", models[0].AwsAccountID.ValueString())
		assert.Equal(t, "accepted", models[0].Status.ValueString())
		assert.Equal(t, "2026-01-01T00:00:00Z", models[0].SharedDate.ValueString())

		assert.Equal(t, int64(20), models[1].ID.ValueInt64())
		assert.False(t, models[1].SharedDate.IsNull())
		assert.Equal(t, "", models[1].SharedDate.ValueString())
	})
}
