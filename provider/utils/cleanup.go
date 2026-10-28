package utils

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

const (
	creationPlanDeleteAttempts   = 3
	creationPlanDeleteRetryDelay = 2 * time.Second
)

// DeleteSubscriptionDatabases deletes all databases in a subscription. The Redis Cloud API requires at
// least one database in the subscription creation plan, but Terraform manages databases as separate
// resources. This function is called during Create and UpdateCmk to clean up those creation-plan
// placeholder databases before Terraform creates user-configured ones.
func DeleteSubscriptionDatabases(ctx context.Context, subId int, api *client.ApiClient, timeout time.Duration) diag.Diagnostics {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// There is a timing issue where the subscription is marked as active before the creation-plan
	// databases are visible via the List API. This sleep works around it.
	if err := waitForCleanupRetry(ctx, 30*time.Second); err != nil {
		return diag.FromErr(err)
	}
	if err := WaitForSubscriptionToBeActive(ctx, subId, api, timeout); err != nil {
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
		if err := WaitForDatabaseToBeActive(ctx, subId, dbId, api, timeout); err != nil {
			var notFound *databases.NotFound
			if errors.As(err, &notFound) {
				continue
			}
			return diag.FromErr(fmt.Errorf("failed waiting for database %d to be active: %w", dbId, err))
		}

		log.Printf("[DEBUG] Deleting database %d in subscription %d", dbId, subId)
		if err := deleteCreationPlanDatabase(ctx, subId, dbId, api.Client.Database, creationPlanDeleteRetryDelay); err != nil {
			return diag.FromErr(fmt.Errorf("failed to delete database %d in subscription %d: %w", dbId, subId, err))
		}
	}
	if dbList.Err() != nil {
		return diag.FromErr(fmt.Errorf("failed to list databases in subscription %d: %w", subId, dbList.Err()))
	}

	if err := WaitForSubscriptionToBeActive(ctx, subId, api, timeout); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

type databaseCleanupAPI interface {
	Get(context.Context, int, int) (*databases.Database, error)
	Delete(context.Context, int, int) error
}

func deleteCreationPlanDatabase(ctx context.Context, subId, dbId int, api databaseCleanupAPI, retryDelay time.Duration) error {
	attempts := 0
	deletionAccepted := false
	var lastDeleteErr error
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		database, err := api.Get(ctx, subId, dbId)
		var notFound *databases.NotFound
		if errors.As(err, &notFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to reconcile database deletion: %w", err)
		}
		if database == nil {
			return fmt.Errorf("database %d returned no state", dbId)
		}

		status := redis.StringValue(database.Status)
		switch {
		case deletionAccepted, status == "delete-pending", status == "deleting", status == "deleted":
			// An accepted deletion may still be visible. Wait for absence without resubmitting it.
		case status == databases.StatusActive:
			if attempts == creationPlanDeleteAttempts {
				return fmt.Errorf("database deletion failed after %d attempts: %w", attempts, lastDeleteErr)
			}
			attempts++
			err = api.Delete(ctx, subId, dbId)
			if err == nil {
				deletionAccepted = true
			} else {
				if !isRetryableCleanupError(err) {
					return err
				}
				lastDeleteErr = err
				log.Printf("[WARN] Creation-plan database %d cleanup in subscription %d failed on attempt %d; reconciling state", dbId, subId, attempts)
			}
		default:
			return fmt.Errorf("database %d has unexpected status %q during deletion", dbId, status)
		}

		delay := retryDelay
		if attempts > 1 {
			delay *= 2
		}
		if err := waitForCleanupRetry(ctx, delay); err != nil {
			return err
		}
	}
}

func isRetryableCleanupError(err error) bool {
	// The SDK's asynchronous task errors expose their status through this interface.
	var apiError interface{ StatusCode() string }
	if !errors.As(err, &apiError) {
		return false
	}
	status, parseErr := strconv.Atoi(apiError.StatusCode())
	if parseErr != nil {
		return false
	}
	switch status {
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func waitForCleanupRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
