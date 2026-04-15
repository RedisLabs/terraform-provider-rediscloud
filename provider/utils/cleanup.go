package utils

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// DeleteCreationPlanDatabases deletes all databases in a subscription. The Redis Cloud API requires
// at least one database in the subscription creation plan, but Terraform manages databases as
// separate resources. This function cleans up those creation-plan databases after the subscription
// becomes active.
//
// It is called from Create (normal flow) and UpdateCmk (after CMK activation).
func DeleteCreationPlanDatabases(ctx context.Context, subId int, api *client.ApiClient) diag.Diagnostics {
	// There is a timing issue where the subscription is marked as active before the creation-plan
	// databases are visible via the List API. This sleep works around it.
	time.Sleep(30 * time.Second) //lintignore:R018
	if err := WaitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}

	dbList := api.Client.Database.List(ctx, subId)

	for dbList.Next() {
		if dbList.Value().ID == nil {
			log.Printf("[WARN] Skipping database with nil ID in subscription %d", subId)
			continue
		}
		dbId := *dbList.Value().ID

		log.Printf("[DEBUG] Waiting for database %d in subscription %d to be active before deletion", dbId, subId)
		if err := WaitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
			return diag.FromErr(fmt.Errorf("failed waiting for database %d to be active: %w", dbId, err))
		}

		log.Printf("[DEBUG] Deleting creation-plan database %d in subscription %d", dbId, subId)
		if err := api.Client.Database.Delete(ctx, subId, dbId); err != nil {
			return diag.FromErr(fmt.Errorf("failed to delete creation-plan database %d: %w", dbId, err))
		}
	}
	if dbList.Err() != nil {
		return diag.FromErr(fmt.Errorf("failed to list databases in subscription %d: %w", subId, dbList.Err()))
	}

	if err := WaitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
