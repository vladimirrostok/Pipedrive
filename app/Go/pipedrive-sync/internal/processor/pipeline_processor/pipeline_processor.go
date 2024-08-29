package pipeline_processor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"pipedrive-sync/pipedrive-sync/internal/model"
	"pipedrive-sync/pipedrive-sync/internal/processor"
	"sync"
)

// We can use this struct to implement the DataProcessor interface and set limits for the data processing.
type PipelineProcessor struct {
	API_KEY                  string
	CustomCustomerIDFieldKey string
	CustomOrderIDFieldKey    string
	HttpWorkerPool           HttpWorkerPool
	ErrChan                  chan error
	MaxSize                  int
}

func (p PipelineProcessor) ProcessOrders(data <-chan model.Order) (chan bool, chan processor.Result) {
	// This is left empty, can be implemented if needed.
	// For now it is not used in the task so it just implements the interface to be more "generic" codebase.
	return nil, nil
}

func (p PipelineProcessor) ProcessPayments(data <-chan model.Payment) (chan bool, chan processor.Result) {
	// This is left empty, can be implemented if needed.
	// For now it is not used in the task so it just implements the interface to be more "generic" codebase.
	return nil, nil
}

func (p PipelineProcessor) ProcessCustomers(data <-chan model.Customer) (chan bool, chan processor.Result) {
	// We return the results channel immediately so we run processing in the goroutine.
	done := make(chan bool)
	wg := sync.WaitGroup{}

	go func() {
		for d := range data {
			wg.Add(1)
			go func() {
				// Prepare the data for the request.
				values := map[string]string{
					"name":                     d.FirstName,
					"last_name":                d.LastName,
					p.CustomCustomerIDFieldKey: d.ID,
				}

				jsonValue, err := json.Marshal(values)
				if err != nil {
					p.ErrChan <- fmt.Errorf("error encoding JSON request %s ", err)
				}

				// Create a new HTTP request with the context, so we can cancel all the ongoing requests if needed.
				req, err := http.NewRequestWithContext(p.HttpWorkerPool.Ctx, "POST", "https://api.pipedrive.com/v1/persons", bytes.NewBuffer(jsonValue))
				if err != nil {
					p.ErrChan <- fmt.Errorf("error in the customer worker job %s %s", d.ID, err)
				}

				// Set the headers for the request.
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/json")

				// Add auth query parameter to the request.
				q := req.URL.Query()
				q.Add("api_token", p.API_KEY)
				req.URL.RawQuery = q.Encode()

				// Push job to the worker pool.
				p.HttpWorkerPool.addCustomerJob(Job{ID: "Add customer: " + d.ID, Request: req})
				wg.Done()
			}()
		}
		wg.Wait()
		close(done)
	}()

	return done, p.HttpWorkerPool.CustomerResults
}

func (p PipelineProcessor) ProcessDeals(data <-chan model.Deal) (chan bool, chan processor.Result) {
	// We return the results channel immediately so we run processing in the goroutine.
	done := make(chan bool)
	wg := sync.WaitGroup{}

	go func() {
		for d := range data {
			wg.Add(1)
			go func() {
				// Prepare the data for the request.
				// Task says: Deals should have a Title of “LastName FirstName”.
				// ID was fixed in the previous step.
				values := map[string]string{
					"title":                 d.CustomerLastName + " " + d.CustomerFirstName,
					"value":                 d.OrderValue,
					"person_id":             d.CustomerID,
					"add_time":              d.OrderDate,
					p.CustomOrderIDFieldKey: d.OrderID,
				}

				jsonValue, err := json.Marshal(values)
				if err != nil {
					p.ErrChan <- fmt.Errorf("error encoding JSON request %s ", err)
				}

				// Create a new HTTP request with the context, so we can cancel all the ongoing requests if needed.
				req, err := http.NewRequestWithContext(p.HttpWorkerPool.Ctx, "POST", "https://api.pipedrive.com/v1/deals", bytes.NewBuffer(jsonValue))
				if err != nil {
					p.ErrChan <- fmt.Errorf("error in the deal worker job %s %s", (d.CustomerLastName + " " + d.CustomerFirstName), err)
				}

				// Set the headers for the request.
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/json")

				// Add auth query parameter to the request.
				q := req.URL.Query()
				q.Add("api_token", p.API_KEY)
				req.URL.RawQuery = q.Encode()

				// Push job to the worker pool.
				p.HttpWorkerPool.addDealJob(Job{ID: "Add deal: " + (d.CustomerLastName + " " + d.CustomerFirstName), Request: req})
				wg.Done()
			}()
		}
		wg.Wait()
		close(done)
	}()

	return done, p.HttpWorkerPool.DealResults
}

func (p PipelineProcessor) ProcessOutOfSyncDeals(data <-chan model.Deal) (chan bool, chan processor.Result) {
	// We return the results channel immediately so we run processing in the goroutine.
	done := make(chan bool)
	wg := sync.WaitGroup{}

	go func() {
		for d := range data {
			wg.Add(1)
			go func() {
				// Prepare the data for the request.
				values := map[string]string{
					"value": d.OrderValue,
				}

				jsonValue, err := json.Marshal(values)
				if err != nil {
					p.ErrChan <- fmt.Errorf("error encoding JSON request %s ", err)
				}

				// Create a new HTTP request with the context, so we can cancel all the ongoing requests if needed.
				req, err := http.NewRequestWithContext(p.HttpWorkerPool.Ctx, "PUT", "https://api.pipedrive.com/v1/deals/"+d.OrderID, bytes.NewBuffer(jsonValue))
				if err != nil {
					p.ErrChan <- fmt.Errorf("error in the patch deal worker job %s %s", (d.CustomerLastName + " " + d.CustomerFirstName), err)
				}

				// Set the headers for the request.
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/json")

				// Add auth query parameter to the request.
				q := req.URL.Query()
				q.Add("api_token", p.API_KEY)
				req.URL.RawQuery = q.Encode()

				// Push job to the worker pool.
				p.HttpWorkerPool.addPatchDealJob(Job{ID: "Patch deal : " + (d.CustomerLastName + " " + d.CustomerFirstName), Request: req})
				wg.Done()
			}()
		}
		wg.Wait()
		close(done)
	}()

	return done, p.HttpWorkerPool.PatchDealResults
}
