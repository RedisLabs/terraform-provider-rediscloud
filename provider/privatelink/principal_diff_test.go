package privatelink

import (
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/stretchr/testify/assert"
)

func TestPrincipalChanges(t *testing.T) {
	tests := map[string]struct {
		apiPrincipals    []*pl.PrivateLinkPrincipal
		tfPrincipals     []pl.PrivateLinkPrincipal
		principalsToAdd  []pl.CreatePrivateLinkPrincipal
		principalsToDrop []pl.PrivateLinkPrincipal
	}{
		"add first principal": {
			apiPrincipals:   apiPrincipalList(),
			tfPrincipals:    terraformPrincipalList("A"),
			principalsToAdd: createPrincipalList("A"),
		},
		"add principal while retaining existing principal": {
			apiPrincipals:   apiPrincipalList("A"),
			tfPrincipals:    terraformPrincipalList("A", "B"),
			principalsToAdd: createPrincipalList("B"),
		},
		"remove principal while retaining existing principal": {
			apiPrincipals:    apiPrincipalList("A", "B"),
			tfPrincipals:     terraformPrincipalList("B"),
			principalsToDrop: terraformPrincipalList("A"),
		},
		"retain all principals": {
			apiPrincipals: apiPrincipalList("A", "B"),
			tfPrincipals:  terraformPrincipalList("A", "B"),
		},
		"add principal while retaining multiple existing principals": {
			apiPrincipals:   apiPrincipalList("A", "B"),
			tfPrincipals:    terraformPrincipalList("A", "B", "C"),
			principalsToAdd: createPrincipalList("C"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.principalsToAdd, findPrincipalsToCreate(test.apiPrincipals, test.tfPrincipals),
				"creation diff must contain exactly the principals present in Terraform configuration but absent from API state")
			assert.Equal(t, test.principalsToDrop, findPrincipalsToDelete(test.apiPrincipals, test.tfPrincipals),
				"deletion diff must contain exactly the principals present in API state but absent from Terraform configuration")
		})
	}
}

// apiPrincipalList builds the pointer-backed shape returned by the API. Each call
// creates independent principal fields so API and Terraform fixtures do not share
// pointer addresses.
func apiPrincipalList(principals ...string) []*pl.PrivateLinkPrincipal {
	values := terraformPrincipalList(principals...)
	result := make([]*pl.PrivateLinkPrincipal, len(values))
	for i := range values {
		result[i] = &values[i]
	}
	return result
}

// terraformPrincipalList builds the value-backed shape derived from Terraform
// configuration. Each call creates independent principal fields so tests compare
// equal strings stored at different addresses.
func terraformPrincipalList(principals ...string) []pl.PrivateLinkPrincipal {
	result := make([]pl.PrivateLinkPrincipal, 0, len(principals))
	for _, principal := range principals {
		result = append(result, privateLinkPrincipal(principal))
	}
	return result
}

func createPrincipalList(principals ...string) []pl.CreatePrivateLinkPrincipal {
	result := make([]pl.CreatePrivateLinkPrincipal, 0, len(principals))
	for _, principal := range principals {
		result = append(result, pl.CreatePrivateLinkPrincipal{
			Principal:      redis.String(principal),
			PrincipalType:  redis.String("aws_account"),
			PrincipalAlias: redis.String("account " + principal),
		})
	}
	return result
}

func privateLinkPrincipal(principal string) pl.PrivateLinkPrincipal {
	return pl.PrivateLinkPrincipal{
		Principal: redis.String(principal),
		Type:      redis.String("aws_account"),
		Alias:     redis.String("account " + principal),
	}
}
