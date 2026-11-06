package pro

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func (r *proDatabaseResource) createDatabase(ctx context.Context, plan *ProDatabaseModel, diagnostics *diag.Diagnostics, timeout time.Duration) {
	if r.api == nil {
		diagnostics.AddError("Unconfigured resource", "The pro database resource has no API client.")
		return
	}
	if plan.SubscriptionID.IsNull() || plan.SubscriptionID.IsUnknown() {
		diagnostics.AddError("Missing subscription_id", "subscription_id must be known before the database can be created.")
		return
	}
	subID := int(plan.SubscriptionID.ValueInt64())
	utils.SubscriptionMutex.Lock(subID)
	locked := true
	defer func() {
		if locked {
			utils.SubscriptionMutex.Unlock(subID)
		}
	}()

	create, diags := buildCreateDatabase(ctx, *plan)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return
	}
	version, _ := knownString(plan.RedisVersion)
	count, countDiags := moduleCount(ctx, plan.Modules)
	diagnostics.Append(countDiags...)
	if ShouldWarnRedis8Modules(version, count > 0) {
		diagnostics.Append(redis8ModulesWarning(version))
	}

	if err := r.waitForSubscription(ctx, subID, timeout); err != nil {
		diagnostics.AddError("Subscription not active", err.Error())
		return
	}
	dbID, err := r.api.Create(ctx, subID, create)
	if err != nil {
		diagnostics.AddError("Failed to create database", err.Error())
		return
	}
	plan.ID = types.StringValue(utils.BuildResourceId(subID, dbID))
	plan.DbID = types.Int64Value(int64(dbID))

	if err := r.waitForDatabase(ctx, subID, dbID, timeout); err != nil {
		diagnostics.AddError("Database failed to become active", r.statusDetail(ctx, subID, dbID, err))
		return
	}
	if err := r.waitForSubscription(ctx, subID, timeout); err != nil {
		diagnostics.AddError("Subscription failed to become active", r.statusDetail(ctx, subID, dbID, err))
		return
	}

	utils.SubscriptionMutex.Unlock(subID)
	locked = false

	// Attributes the create API does not accept are applied by update.
	r.updateDatabase(ctx, plan, nil, diagnostics, timeout)
	if diagnostics.HasError() {
		return
	}
	r.readDatabase(ctx, plan, diagnostics, timeout)
}

func (r *proDatabaseResource) readDatabase(ctx context.Context, state *ProDatabaseModel, diagnostics *diag.Diagnostics, timeout time.Duration) bool {
	if r.api == nil {
		diagnostics.AddError("Unconfigured resource", "The pro database resource has no API client.")
		return false
	}
	subID, dbID, err := ToDatabaseId(state.ID.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid resource ID", err.Error())
		return false
	}
	if subID == 0 && !state.SubscriptionID.IsNull() && !state.SubscriptionID.IsUnknown() {
		subID = int(state.SubscriptionID.ValueInt64())
	}

	db, err := r.api.Get(ctx, subID, dbID)
	if err != nil {
		var notFound *databases.NotFound
		if errors.As(err, &notFound) {
			return true
		}
		diagnostics.AddError("Failed to read database", err.Error())
		return false
	}

	var access *bool
	resolveDefaults := false
	apiSource := []string{}
	if db.Security != nil {
		apiSource = pointerStrings(db.Security.SourceIPs)
	}
	stateSource, sourceDiags := stringSet(ctx, state.SourceIPs)
	diagnostics.Append(sourceDiags...)
	_, needsDefault := resolveReadSourceIPs(apiSource, stateSource)
	if needsDefault {
		if err := r.waitForSubscription(ctx, subID, timeout); err != nil {
			diagnostics.AddError("Subscription not active", err.Error())
			return false
		}
		access, err = r.api.SubscriptionPublicEndpointAccess(ctx, subID)
		if err != nil {
			diagnostics.AddError("Failed to read subscription", err.Error())
			return false
		}
		resolveDefaults = true
	}

	tags, err := r.api.GetTags(ctx, subID, dbID)
	if err != nil {
		diagnostics.AddError("Failed to read database tags", err.Error())
		return false
	}

	updated, diags := applyProDatabaseRead(ctx, *state, db, access, tags, resolveDefaults)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return false
	}
	*state = updated
	return false
}

func (r *proDatabaseResource) updateDatabase(ctx context.Context, plan *ProDatabaseModel, prior *ProDatabaseModel, diagnostics *diag.Diagnostics, timeout time.Duration) {
	if r.api == nil {
		diagnostics.AddError("Unconfigured resource", "The pro database resource has no API client.")
		return
	}
	subID, dbID, err := ToDatabaseId(plan.ID.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}
	if subID == 0 && !plan.SubscriptionID.IsNull() && !plan.SubscriptionID.IsUnknown() {
		subID = int(plan.SubscriptionID.ValueInt64())
	}

	utils.SubscriptionMutex.Lock(subID)
	defer utils.SubscriptionMutex.Unlock(subID)

	version, _ := knownString(plan.RedisVersion)
	count, countDiags := moduleCount(ctx, plan.Modules)
	diagnostics.Append(countDiags...)
	if ShouldWarnRedis8Modules(version, count > 0) {
		diagnostics.Append(redis8ModulesWarning(version))
	}

	if err := r.waitForSubscription(ctx, subID, timeout); err != nil {
		diagnostics.AddError("Subscription not active", err.Error())
		return
	}
	if err := r.waitForDatabase(ctx, subID, dbID, timeout); err != nil {
		diagnostics.AddError("Database not active", r.statusDetail(ctx, subID, dbID, err))
		return
	}

	if prior != nil && shouldUpgradeRedisVersion(prior.RedisVersion, plan.RedisVersion, prior.RedisVersionActual) {
		target := plan.RedisVersion.ValueString()
		log.Printf("[INFO] Requesting Redis version change to %s...", target)
		upgradeErr := r.api.UpgradeRedisVersion(ctx, subID, dbID, databases.UpgradeRedisVersion{TargetRedisVersion: redis.String(target)})
		if upgradeErr != nil {
			diagnostics.AddError("Failed to change Redis version", fmt.Sprintf("failed to change Redis version to %s: %v", target, upgradeErr))
			return
		}
		log.Printf("[INFO] Redis version change request to %s accepted by API", target)
		if err := r.waitForDatabase(ctx, subID, dbID, timeout); err != nil {
			diagnostics.AddError("Database failed to become active after Redis version change", r.statusDetail(ctx, subID, dbID, err))
			return
		}
		if err := r.waitForSubscription(ctx, subID, timeout); err != nil {
			diagnostics.AddError("Subscription failed to become active after Redis version change", err.Error())
			return
		}
	}

	var access *bool
	sourceIPs, sourceDiags := stringSet(ctx, plan.SourceIPs)
	diagnostics.Append(sourceDiags...)
	if diagnostics.HasError() {
		return
	}
	if plan.SourceIPs.IsNull() || plan.SourceIPs.IsUnknown() || isDefaultSourceIPStrings(sourceIPs) {
		access, err = r.api.SubscriptionPublicEndpointAccess(ctx, subID)
		if err != nil {
			diagnostics.AddError("Failed to read subscription", err.Error())
			return
		}
	}

	update, updateDiags := buildUpdateDatabase(ctx, *plan, prior, access)
	diagnostics.Append(updateDiags...)
	if diagnostics.HasError() {
		return
	}
	if err := r.api.Update(ctx, subID, dbID, update); err != nil {
		diagnostics.AddError("Failed to update database", err.Error())
		return
	}
	if err := r.waitForDatabase(ctx, subID, dbID, timeout); err != nil {
		diagnostics.AddError("Database failed to become active", r.statusDetail(ctx, subID, dbID, err))
		return
	}
	if err := r.waitForSubscription(ctx, subID, timeout); err != nil {
		diagnostics.AddError("Subscription failed to become active", err.Error())
		return
	}

	tags, tagDiags := stringMap(ctx, plan.Tags)
	diagnostics.Append(tagDiags...)
	if diagnostics.HasError() {
		return
	}
	if err := r.api.PutTags(ctx, subID, dbID, tags); err != nil {
		diagnostics.AddError("Failed to write database tags", err.Error())
	}
}

func (r *proDatabaseResource) deleteDatabase(ctx context.Context, state *ProDatabaseModel, diagnostics *diag.Diagnostics, timeout time.Duration) {
	if r.api == nil {
		diagnostics.AddError("Unconfigured resource", "The pro database resource has no API client.")
		return
	}
	if state.SubscriptionID.IsNull() || state.SubscriptionID.IsUnknown() {
		diagnostics.AddError("Missing subscription_id", "subscription_id must be known before the database can be deleted.")
		return
	}
	subID := int(state.SubscriptionID.ValueInt64())
	_, dbID, err := ToDatabaseId(state.ID.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}

	utils.SubscriptionMutex.Lock(subID)
	defer utils.SubscriptionMutex.Unlock(subID)

	if err := r.waitForSubscription(ctx, subID, timeout); err != nil {
		diagnostics.AddError("Subscription not active", err.Error())
		return
	}
	if err := r.waitForDatabase(ctx, subID, dbID, timeout); err != nil {
		diagnostics.AddError("Database not active", r.statusDetail(ctx, subID, dbID, err))
		return
	}
	if err := r.api.Delete(ctx, subID, dbID); err != nil {
		diagnostics.AddError("Failed to delete database", err.Error())
		return
	}
	if err := r.waitForSubscription(ctx, subID, timeout); err != nil {
		diagnostics.AddError("Subscription failed to become active", err.Error())
	}
}

func (r *proDatabaseResource) statusDetail(ctx context.Context, subscriptionID, databaseID int, err error) string {
	db, getErr := r.api.Get(ctx, subscriptionID, databaseID)
	if getErr != nil || db == nil {
		return err.Error()
	}
	return observedStatusDetail(db.Status, err)
}
