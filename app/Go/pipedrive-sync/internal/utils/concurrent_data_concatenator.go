package utils

import (
	"encoding/csv"
	"os"
	"pipedrive-sync/pipedrive-sync/internal/model"
	"pipedrive-sync/pipedrive-sync/internal/reader"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

// There are multiple payments for the same order, we need to sum them up to set the single "Value" for the order in Pipedrive API.
func SumPayments(dataReader reader.DataReader, inputFile, outputFile string, wg *sync.WaitGroup, errChan chan error) error {
	// Create a file to write the data to.
	wg.Add(1)
	defer wg.Done()

	out, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	writer := csv.NewWriter(out)
	// Always flush contents of the writer and also clsoe the file afterwards.
	defer writer.Flush()
	defer out.Close()

	// Write the header to the output CSV, we're re-assembling the data in a logical order.
	err = writer.Write([]string{"id", "order_id", "payment_method", "amount"})
	if err != nil {
		return err
	}

	// Measure the performance.
	start := time.Now()

	// Synchronize the writer.
	writeMutex := sync.Mutex{}

	wgPayments := sync.WaitGroup{}

	/*
		By reading data in batches we can process it much more efficiently,
		Search every payment => every payment, this takes a lot of reading,
		instead we copy huge batches of data from buffered channel, reduce memory allocations,
		and also make this task CPU bound instead of IO bound. (We buffered the file in reader too).

		This is heavy CPU blocking task, we need to process the data in order.
		We already read batches from the buffered channels, so we can't speed it up much more.
		So we parallelize this task and use all available cores to process the data.
	*/
	// Iterate over the payments and sum the total amount for each order.
	// Read batch of payments.
	for batchPayments := range dataReader.ReadPayments("raw_payments.csv", 100) {
		wgPayments.Add(1)

		// Run payments batch processing in parallel.
		go func() {
			wgPaymentFromBatch := sync.WaitGroup{}
			defer wgPayments.Done()
			// Store the processed payments in a map to sum them up.
			batchProcessed := make(map[string]model.Payment)

			// Read payment and use it to sum the total amount for the order.
			for _, payment := range batchPayments {
				wgPaymentFromBatch.Add(1)

				sumTotal := 0

				// Run processing for every payment in parallel to find corresponding payments with similar OrderID.
				// Run new goroutine with reader, doing much more goroutines inside here in parallel is not really efficient here.
				go func() {
					defer wgPaymentFromBatch.Done()

					// Read batch of payments.
					for batch := range dataReader.ReadPayments("raw_payments.csv", 100) {
						// Process every payment and see if there are multiple payments for same order, sum them up.
						for _, i := range batch {
							if payment.OrderID == i.OrderID {
								sumTotal += payment.Amount + i.Amount

								batchProcessed[i.OrderID] = model.Payment{
									ID:            i.ID,
									OrderID:       i.OrderID,
									PaymentMethod: i.PaymentMethod,
									Amount:        sumTotal,
								}
							}
						}
					}

				}()
				wgPaymentFromBatch.Wait()
			}

			// On the last iteration we have empty batch, so skip this case.
			if len(batchProcessed) > 0 {
				for _, summedPayment := range batchProcessed {
					err := writer.Write([]string{
						summedPayment.ID,
						summedPayment.OrderID,
						summedPayment.PaymentMethod,
						strconv.Itoa(summedPayment.Amount),
					})
					if err != nil {
						errChan <- err
					}
				}
				batchProcessed = nil // Help GC to clean up the memory.

			}
		}()
		wgPayments.Wait()

		// Protect the writer from concurrent writes otherwise it might return "short write" error.
		writeMutex.Lock()
		// Flush after each big batch (on big data we should flush even more often).
		writer.Flush()
		writeMutex.Unlock()
	}
	wgPayments.Wait()
	zap.S().Infof("Processing time for payments sum file %s ", time.Since(start))
	return nil
}

// Get data in order and go through it all once to connect the data logically.
func ConcatenateData(errChan chan error, dataReader reader.DataReader, outputFile []string, combinedFile, combinedPaymentsFile string) error {
	// Create a file to write the data to.
	out, err := os.Create(combinedFile)
	if err != nil {
		return err
	}

	writer := csv.NewWriter(out)

	// Always flush contents of the writer and also clsoe the file afterwards.
	defer writer.Flush()
	defer out.Close()

	// Write the header to the output CSV, we're re-assembling the data in logical order.
	err = writer.Write([]string{"CustomerID", "OrderID", "CustomerFirstName", "CustomerLastName", "OrderDate", "Amount"})
	if err != nil {
		return err
	}

	// Measure the performance.
	start := time.Now()

	// Tune the buffer sizes for the channels.
	// These are optimal settings.
	customersBuffSize := 20
	ordersBuffSize := 20
	paymentsBuffSize := 1000 // We process the payments in batches every time for each order so let's read a lot in one go.

	// Synchronize the writer.
	writeMutex := sync.Mutex{}
	var wgCustomers sync.WaitGroup

	/*
		By reading data in batches we can process it much more efficiently,
		Search every payment => every order => every customer, this takes a lot of reading,
		instead we copy huge batches of data from buffered channel, reduce memory allocations,
		and also make this task CPU bound instead of IO bound. (We buffered the file in reader too).

		This is heavy CPU blocking task, we need to process the data in order.
		We already read batches from the buffered channels, so we can't speed it up much more.
		So we parallelize this task and use all available cores to process the data.
	*/
	// Read customers in batches.
	for batchCustomers := range dataReader.ReadCustomers(outputFile[0], customersBuffSize) {
		wgCustomers.Add(1)
		// Run customers batch processing in parallel.
		go func() {
			wgOrders := sync.WaitGroup{}
			defer wgCustomers.Done()

			// Read orders in batches.
			for batchOrders := range dataReader.ReadOrders(outputFile[1], ordersBuffSize) {
				wgOrders.Add(1)

				// Run orders batch processing in parallel.
				go func() {
					wgProcess := sync.WaitGroup{}
					defer wgOrders.Done()

					// Run payment batch processing in parallel.
					for batchPayments := range dataReader.ReadPayments(combinedPaymentsFile, paymentsBuffSize) {
						wgProcess.Add(1)

						// Run processing cutomer<->order<->summmed_payments in parallel.
						go func() {
							defer wgProcess.Done()
							processCustomersBatch(batchCustomers, batchOrders, batchPayments, writer, errChan)
						}()
						wgProcess.Wait()
					}
				}()
				wgOrders.Wait()
			}
			// Protect the writer from concurrent writes otherwise it might return "short write" error.
			writeMutex.Lock()
			// Flush after each big batch per customer (on big data we should flush even more often).
			writer.Flush()
			writeMutex.Unlock()
		}()
		wgCustomers.Wait()
	}

	zap.S().Infof("Processing time for deals file %s ", time.Since(start))
	return nil
}

func processCustomersBatch(batchCustomers []model.Customer, batchOrders []model.Order, batchPayments []model.Payment, writer *csv.Writer, errChan chan error) {
	// NB! Running this task in parallel creates too much overhead, so it's faster without parallelism here.
	// wg := sync.WaitGroup{}
	for _, customer := range batchCustomers {
		// wg.Add(1)
		// Run batch of orders processing in parallel.
		// go func() {
		// defer wg.Done()
		processOrdersBatch(batchOrders, batchPayments, customer, writer, errChan)
		// }()
		// wg.Wait()
	}
}

func processOrdersBatch(batchOrders []model.Order, batchPayments []model.Payment, customer model.Customer, writer *csv.Writer, errChan chan error) {
	// NB! Running this task in parallel creates too much overhead, so it's faster without parallelism here.
	// wg := sync.WaitGroup{}
	for _, order := range batchOrders {
		// wg.Add(1)
		// Run batch of orders processing in parallel.
		// go func() {
		// defer wg.Done()

		// Running order processing in parallel creates too much overhead, so it's faster without parallelism here.
		processPaymentsBatch(batchPayments, order, customer, writer, errChan)
		// }()
		// wg.Wait()
	}
}

// Process total amount for the order in a batch, we can have multiple payments for same order.
func processPaymentsBatch(batchPayments []model.Payment, order model.Order, customer model.Customer, writer *csv.Writer, errChan chan error) {
	for _, payment := range batchPayments {
		if order.UserID == customer.ID && order.ID == payment.OrderID {
			err := writer.Write([]string{
				customer.ID,
				order.ID,
				customer.FirstName,
				customer.LastName,
				order.OrderDate,
				strconv.Itoa(payment.Amount),
			})
			if err != nil {
				errChan <- err
			}
		}
	}
}
