package psc_test

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pscapi "github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/psc"
)

func TestEndpointsFromAPI(t *testing.T) {
	ctx := context.Background()
	endpoints := []*pscapi.PrivateServiceConnectEndpoint{
		{
			ID:                     redis.Int(1),
			GCPProjectID:           redis.String("project"),
			GCPVPCName:             redis.String("vpc"),
			GCPVPCSubnetName:       redis.String("subnet"),
			EndpointConnectionName: redis.String("redis-1"),
			Status:                 redis.String(pscapi.EndpointStatusInitialized),
		},
		{ID: redis.Int(2), Status: redis.String(pscapi.EndpointStatusRejected)},
	}
	attachments := map[int][]pscapi.TerraformGCPServiceAttachment{
		1: {{Name: redis.String("attachment"), DNSRecord: redis.String("dns"), IPAddressName: redis.String("ip"), ForwardingRuleName: redis.String("rule")}},
	}

	list, diags := psc.EndpointsFromAPI(ctx, endpoints, attachments)
	require.False(t, diags.HasError(), diags)

	var models []psc.EndpointModel
	require.False(t, list.ElementsAs(ctx, &models, false).HasError())
	require.Len(t, models, 2)

	assert.Equal(t, types.Int64Value(1), models[0].PrivateServiceConnectEndpointID)
	assert.Equal(t, types.StringValue("project"), models[0].GCPProjectID)
	assert.Equal(t, types.StringValue("redis-1"), models[0].EndpointConnectionName)
	var first []psc.ServiceAttachmentModel
	require.False(t, models[0].ServiceAttachments.ElementsAs(ctx, &first, false).HasError())
	assert.Equal(t, []psc.ServiceAttachmentModel{{
		Name:               types.StringValue("attachment"),
		DNSRecord:          types.StringValue("dns"),
		IPAddressName:      types.StringValue("ip"),
		ForwardingRuleName: types.StringValue("rule"),
	}}, first)

	assert.Equal(t, types.StringValue(""), models[1].GCPProjectID, "SDKv2 wrote empty strings for missing fields")
	assert.False(t, models[1].ServiceAttachments.IsNull())
	assert.Empty(t, models[1].ServiceAttachments.Elements())
}

func TestEndpointsFromAPIEmpty(t *testing.T) {
	list, diags := psc.EndpointsFromAPI(context.Background(), nil, nil)
	require.False(t, diags.HasError())
	assert.False(t, list.IsNull())
	assert.Empty(t, list.Elements())
}
