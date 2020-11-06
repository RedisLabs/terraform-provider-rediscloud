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
