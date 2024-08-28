package processor

import (
	"pipedrive-sync/pipedrive-sync/internal/model"
)

// The DataProcessor interface is used to process the data in custom way.
type DataProcessor interface {
	ProcessPayments(data <-chan model.Payment) (done chan bool, result chan Result)
	ProcessOrders(data <-chan model.Order) (done chan bool, result chan Result)
	ProcessCustomers(data <-chan model.Customer) (done chan bool, result chan Result)
	ProcessDeals(data <-chan model.Deal) (done chan bool, result chan Result)
	ProcessOutOfSyncDeals(data <-chan model.Deal) (done chan bool, result chan Result)
}

type Result struct {
	JobID  string
	Status string
	Error  error
}
