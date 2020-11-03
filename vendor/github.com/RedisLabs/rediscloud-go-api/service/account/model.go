package account

type paymentMethods struct {
	PaymentMethods []PaymentMethod `json:"paymentMethods"`
}

type PaymentMethod struct {
	Id                 int    `json:"id"`
	Type               string `json:"type"`
	CreditCardEndsWith int    `json:"creditCardEndsWith"`
}
