package transitgateway

import (
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataSourceCidrsFromAPI(t *testing.T) {
	ctx := t.Context()
	first := &attachments.Cidr{CidrAddress: redis.String("10.0.0.0/24")}
	second := &attachments.Cidr{CidrAddress: redis.String("10.1.0.0/24")}

	for _, tt := range []struct {
		name string
		api  []*attachments.Cidr
		want []string
	}{
		{"API returns ascending order", []*attachments.Cidr{first, second}, []string{"10.0.0.0/24", "10.1.0.0/24"}},
		{"API returns descending order", []*attachments.Cidr{second, first}, []string{"10.0.0.0/24", "10.1.0.0/24"}},
		{"API returns no CIDRs", nil, []string{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := dataSourceCidrsFromAPI(ctx, tt.api)
			require.False(t, diags.HasError())
			want, diags := types.ListValueFrom(ctx, types.StringType, tt.want)
			require.False(t, diags.HasError())
			assert.True(t, got.Equal(want), "got %s, want %s", got, want)
		})
	}
}

func TestCidrsFromAPIKeepsAPIOrder(t *testing.T) {
	ctx := t.Context()
	got, diags := CidrsFromAPI(ctx, []*attachments.Cidr{
		{CidrAddress: redis.String("10.1.0.0/24")},
		{CidrAddress: redis.String("10.0.0.0/24")},
	})
	require.False(t, diags.HasError())
	var addresses []string
	require.False(t, got.ElementsAs(ctx, &addresses, false).HasError())
	assert.Equal(t, []string{"10.1.0.0/24", "10.0.0.0/24"}, addresses)
}
