package account

import (
	"context"
)

type HttpClient interface {
	Get(ctx context.Context, name, path string, responseBody interface{}) error
}

type API struct {
	client HttpClient
}

func NewAPI(client HttpClient) *API {
	return &API{client: client}
}

// ListPaymentMethods will return the list of available payment methods.
func (a *API) ListPaymentMethods(ctx context.Context) ([]*PaymentMethod, error) {
	var body paymentMethods
	if err := a.client.Get(ctx, "list payment methods", "/payment-methods", &body); err != nil {
		return nil, err
	}

	return body.PaymentMethods, nil
}

// ListRegions will return the list of available regions.
func (a *API) ListRegions(ctx context.Context) ([]*Region, error) {
	var body regions
	if err := a.client.Get(ctx, "list regions", "/regions", &body); err != nil {
		return nil, err
	}

	return body.Regions, nil
}

// ListDataPersistence will return the list of available data persistence values.
func (a *API) ListDataPersistence(ctx context.Context) ([]*DataPersistence, error) {
	var body dataPersistence
	if err := a.client.Get(ctx, "list data persistence", "/data-persistence", &body); err != nil {
		return nil, err
	}

	return body.DataPersistence, nil
}

// ListDataModules will return the list of available data modules that can be applied to a database.
func (a *API) ListDatabaseModules(ctx context.Context) ([]*DatabaseModule, error) {
	var body databaseModules
	if err := a.client.Get(ctx, "list database modules", "/database-modules", &body); err != nil {
		return nil, err
	}

	return body.DatabaseModules, nil
}
