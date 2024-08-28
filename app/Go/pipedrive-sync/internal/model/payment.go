package model

type Payment struct {
	ID            string
	OrderID       string
	PaymentMethod string
	Amount        int
}
