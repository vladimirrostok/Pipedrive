package reader

import "pipedrive-sync/pipedrive-sync/internal/model"

// The DataReader interface is used to read and stream data from the specified file.
type DataReader interface {
	ReadCustomers(filepath string, batchSize int) chan []model.Customer
	ReadOrders(filepath string, batchSize int) chan []model.Order
	ReadPayments(filepath string, batchSize int) chan []model.Payment
	ReadDeals(filepath string, batchSize int) chan []model.Deal
}
