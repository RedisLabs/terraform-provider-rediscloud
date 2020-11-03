package account

import (
	"context"
)

type HttpClient interface {
	Get(ctx context.Context, name, path string, responseBody interface{}) error
}

type Api struct {
	client HttpClient
}

func NewApi(client HttpClient) *Api {
	return &Api{client: client}
}

// ListPaymentMethods will return the list of available payment methods.
func (a *Api) ListPaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	var body paymentMethods
	if err := a.client.Get(ctx, "list payment methods", "/payment-methods", &body); err != nil {
		return nil, err
	}

	return body.PaymentMethods, nil
}
