package privatelink

import (
	"context"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// ConnectionModel represents a connection attached to a PrivateLink.
type ConnectionModel struct {
	AssociationID   types.String `tfsdk:"association_id"`
	ConnectionID    types.String `tfsdk:"connection_id"`
	ConnectionType  types.String `tfsdk:"connection_type"`
	OwnerID         types.String `tfsdk:"owner_id"`
	AssociationDate types.String `tfsdk:"association_date"`
}

// DatabaseModel represents a database reachable through a PrivateLink.
type DatabaseModel struct {
	DatabaseID           types.Int64  `tfsdk:"database_id"`
	Port                 types.Int64  `tfsdk:"port"`
	ResourceLinkEndpoint types.String `tfsdk:"resource_link_endpoint"`
}

var (
	connectionElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(ConnectionModel{})}
	databaseElemType   = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(DatabaseModel{})}
)

// PrincipalModelsFromAPI maps API principals as the SDKv2 implementation did, writing "" for missing fields.
func PrincipalModelsFromAPI(principals []*pl.PrivateLinkPrincipal) []PrincipalModel {
	models := make([]PrincipalModel, 0, len(principals))
	for _, principal := range principals {
		//TODO(TF3.0) make principal, principal_type and principal_alias nullable
		models = append(models, PrincipalModel{
			Principal:      types.StringValue(redis.StringValue(principal.Principal)),
			PrincipalType:  types.StringValue(redis.StringValue(principal.Type)),
			PrincipalAlias: types.StringValue(redis.StringValue(principal.Alias)),
		})
	}
	return models
}

// PrincipalsFromAPI returns the computed principal set exposed by the data sources.
func PrincipalsFromAPI(ctx context.Context, principals []*pl.PrivateLinkPrincipal) (PrincipalSetValue, diag.Diagnostics) {
	return NewPrincipalSet(ctx, PrincipalModelsFromAPI(principals))
}

// ConnectionsFromAPI returns the computed connection set of a PrivateLink.
func ConnectionsFromAPI(ctx context.Context, connections []*pl.PrivateLinkConnection) (types.Set, diag.Diagnostics) {
	models := make([]ConnectionModel, 0, len(connections))
	for _, connection := range connections {
		//TODO(TF3.0) make connection fields nullable
		models = append(models, ConnectionModel{
			AssociationID:   types.StringValue(redis.StringValue(connection.AssociationId)),
			ConnectionID:    types.StringValue(redis.StringValue(connection.ConnectionId)),
			ConnectionType:  types.StringValue(redis.StringValue(connection.Type)),
			OwnerID:         types.StringValue(redis.StringValue(connection.OwnerId)),
			AssociationDate: types.StringValue(redis.StringValue(connection.AssociationDate)),
		})
	}
	return types.SetValueFrom(ctx, connectionElemType, models)
}

// DatabasesFromAPI returns the computed database set of a PrivateLink.
func DatabasesFromAPI(ctx context.Context, databases []*pl.PrivateLinkDatabase) (types.Set, diag.Diagnostics) {
	models := make([]DatabaseModel, 0, len(databases))
	for _, database := range databases {
		//TODO(TF3.0) make resource_link_endpoint nullable
		models = append(models, DatabaseModel{
			DatabaseID:           types.Int64Value(int64(redis.IntValue(database.DatabaseId))),
			Port:                 types.Int64Value(int64(redis.IntValue(database.Port))),
			ResourceLinkEndpoint: types.StringValue(redis.StringValue(database.ResourceLinkEndpoint)),
		})
	}
	return types.SetValueFrom(ctx, databaseElemType, models)
}

func createOtherPrincipals(ctx context.Context, api *client.ApiClient, subId int, otherPrincipals []pl.PrivateLinkPrincipal) error {
	if len(otherPrincipals) > 0 {
		for _, principal := range otherPrincipals {
			err := api.Client.PrivateLink.CreatePrincipal(ctx, subId, pl.CreatePrivateLinkPrincipal{
				Principal:      principal.Principal,
				PrincipalType:  principal.Type,
				PrincipalAlias: principal.Alias,
			})

			if err != nil {
				return err
			}
		}
	}
	return nil
}

func findPrincipalsToCreate(apiPrincipals []*pl.PrivateLinkPrincipal, tfPrincipals []pl.PrivateLinkPrincipal) []pl.CreatePrivateLinkPrincipal {
	var principals []pl.PrivateLinkPrincipal

	for _, tfPrincipal := range tfPrincipals {
		found := false
		for _, apiPrincipal := range apiPrincipals {
			if redis.StringValue(tfPrincipal.Principal) == redis.StringValue(apiPrincipal.Principal) {
				found = true
				break
			}
		}
		if !found {
			principals = append(principals, tfPrincipal)
		}
	}

	var createPrincipals []pl.CreatePrivateLinkPrincipal
	for _, principal := range principals {
		createPrincipal := pl.CreatePrivateLinkPrincipal{
			Principal:      principal.Principal,
			PrincipalType:  principal.Type,
			PrincipalAlias: principal.Alias,
		}
		createPrincipals = append(createPrincipals, createPrincipal)
	}

	return createPrincipals
}

func createPrincipals(ctx context.Context, api *client.ApiClient, subscriptionId int, principals []pl.CreatePrivateLinkPrincipal) error {
	for _, principal := range principals {
		err := api.Client.PrivateLink.CreatePrincipal(ctx, subscriptionId, principal)
		if err != nil {
			return err
		}
	}
	return nil
}

func findPrincipalsToDelete(apiPrincipals []*pl.PrivateLinkPrincipal, tfPrincipals []pl.PrivateLinkPrincipal) []pl.PrivateLinkPrincipal {
	var result []pl.PrivateLinkPrincipal

	for _, apiPrincipal := range apiPrincipals {
		found := false
		for _, tfPrincipal := range tfPrincipals {
			if redis.StringValue(apiPrincipal.Principal) == redis.StringValue(tfPrincipal.Principal) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, *apiPrincipal)
		}
	}
	return result
}

func deletePrincipals(ctx context.Context, api *client.ApiClient, subscriptionId int, principals []pl.PrivateLinkPrincipal) error {
	for _, principal := range principals {
		err := api.Client.PrivateLink.DeletePrincipal(ctx, subscriptionId, *principal.Principal)
		if err != nil {
			return err
		}
	}
	return nil
}
