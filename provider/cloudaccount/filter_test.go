package cloudaccount

import (
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/stretchr/testify/assert"
)

func TestFilterCloudAccounts(t *testing.T) {
	internal := &cloud_accounts.CloudAccount{ID: redis.Int(1), Name: redis.String("internal"), Provider: redis.String("AWS")}
	awsProd := &cloud_accounts.CloudAccount{ID: redis.Int(2), Name: redis.String("prod"), Provider: redis.String("AWS")}
	gcpProd := &cloud_accounts.CloudAccount{ID: redis.Int(3), Name: redis.String("prod"), Provider: redis.String("GCP")}
	awsDev := &cloud_accounts.CloudAccount{ID: redis.Int(4), Name: redis.String("dev"), Provider: redis.String("AWS")}

	tests := []struct {
		name    string
		input   []*cloud_accounts.CloudAccount
		filters []caFilterFunc
		want    []*cloud_accounts.CloudAccount
	}{
		{
			name:  "no filters returns all non-nil accounts",
			input: []*cloud_accounts.CloudAccount{internal, awsProd, gcpProd},
			want:  []*cloud_accounts.CloudAccount{internal, awsProd, gcpProd},
		},
		{
			name:  "nil accounts are skipped",
			input: []*cloud_accounts.CloudAccount{nil, awsProd, nil},
			want:  []*cloud_accounts.CloudAccount{awsProd},
		},
		{
			name:    "exclude internal account (id 1)",
			input:   []*cloud_accounts.CloudAccount{internal, awsProd, gcpProd},
			filters: []caFilterFunc{excludeInternalAccountFilter()},
			want:    []*cloud_accounts.CloudAccount{awsProd, gcpProd},
		},
		{
			name:    "single provider filter",
			input:   []*cloud_accounts.CloudAccount{awsProd, gcpProd, awsDev},
			filters: []caFilterFunc{providerTypeFilter("AWS")},
			want:    []*cloud_accounts.CloudAccount{awsProd, awsDev},
		},
		{
			name:    "multiple filters are ANDed (provider and name)",
			input:   []*cloud_accounts.CloudAccount{awsProd, gcpProd, awsDev},
			filters: []caFilterFunc{providerTypeFilter("AWS"), nameFilter("prod")},
			want:    []*cloud_accounts.CloudAccount{awsProd},
		},
		{
			name:    "no matches returns empty",
			input:   []*cloud_accounts.CloudAccount{awsProd, gcpProd},
			filters: []caFilterFunc{providerTypeFilter("Azure")},
			want:    nil,
		},
		{
			name:  "empty input returns empty",
			input: nil,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterCloudAccounts(tt.input, tt.filters)
			assert.Equal(t, tt.want, got)
		})
	}
}
