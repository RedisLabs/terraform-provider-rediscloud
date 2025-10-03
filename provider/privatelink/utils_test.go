package privatelink

import (
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/stretchr/testify/assert"
)

func TestUnitFlattenConnections(t *testing.T) {
	// Test that flattenConnections produces keys that match the schema
	connections := []*pl.PrivateLinkConnection{
		{
			AssociationId:   redis.String("assoc-123"),
			ConnectionId:    redis.String("conn-456"),
			Type:            redis.String("vpc-endpoint"),
			OwnerId:         redis.String("owner-789"),
			AssociationDate: redis.String("2024-01-01"),
		},
	}

	result := flattenConnections(connections)

	assert.Len(t, result, 1)

	conn := result[0]
	assert.Contains(t, conn, "association_id")
	assert.Contains(t, conn, "connection_id")
	assert.Contains(t, conn, "connection_type")
	assert.Contains(t, conn, "owner_id")
	assert.Contains(t, conn, "association_date")

	assert.Equal(t, "assoc-123", conn["association_id"])
	assert.Equal(t, "conn-456", conn["connection_id"])
	assert.Equal(t, "vpc-endpoint", conn["connection_type"])
	assert.Equal(t, "owner-789", conn["owner_id"])
	assert.Equal(t, "2024-01-01", conn["association_date"])
}

func TestUnitFlattenConnectionsEmpty(t *testing.T) {
	result := flattenConnections([]*pl.PrivateLinkConnection{})
	assert.Len(t, result, 0)
}

func TestUnitFlattenDatabases(t *testing.T) {
	databases := []*pl.PrivateLinkDatabase{
		{
			DatabaseId:           redis.Int(12345),
			Port:                 redis.Int(16379),
			ResourceLinkEndpoint: redis.String("endpoint.example.com"),
		},
	}

	result := flattenDatabases(databases)

	assert.Len(t, result, 1)

	db := result[0]
	assert.Contains(t, db, "database_id")
	assert.Contains(t, db, "port")
	assert.Contains(t, db, "resource_link_endpoint")

	assert.Equal(t, 12345, db["database_id"])
	assert.Equal(t, 16379, db["port"])
	assert.Equal(t, "endpoint.example.com", db["resource_link_endpoint"])
}

func TestUnitFlattenPrincipals(t *testing.T) {
	principals := []*pl.PrivateLinkPrincipal{
		{
			Principal: redis.String("arn:aws:iam::123456789012:root"),
			Type:      redis.String("aws_account"),
			Alias:     redis.String("my-account"),
		},
		{
			Principal: redis.String("arn:aws:iam::987654321098:root"),
			Type:      redis.String("aws_account"),
			Alias:     redis.String("another-account"),
		},
	}

	result := flattenPrincipals(principals)

	assert.Len(t, result, 2)

	// Verify keys
	for _, p := range result {
		assert.Contains(t, p, "principal")
		assert.Contains(t, p, "principal_type")
		assert.Contains(t, p, "principal_alias")
	}

	// Results should be sorted by principal
	assert.Equal(t, "arn:aws:iam::123456789012:root", result[0]["principal"])
	assert.Equal(t, "arn:aws:iam::987654321098:root", result[1]["principal"])
}
