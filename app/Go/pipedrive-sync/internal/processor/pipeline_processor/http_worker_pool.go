package pipeline_processor

import (
	"context"
	"fmt"
	"net/http"
	"pipedrive-sync/pipedrive-sync/internal/processor"
	"pipedrive-sync/pipedrive-sync/internal/utils"
	"sync"
	"time"
)

// We use worker pool to control the maximum amount of goroutines.
type HttpWorkerPool struct {
	Ctx context.Context
	// Go handles pool of HTTP connections inside the client so we can use single instance of it.
	client           utils.HttpClient
	errChan          chan error
	CustomerJobs     chan Job
	DealJobs         chan Job
	PatchDealJobs    chan Job
	CustomerResults  chan processor.Result
	DealResults      chan processor.Result
	PatchDealResults chan processor.Result
	wg               *sync.WaitGroup
}

// A Job is a struct that contains the ID and URL of the job.
// We can assign job a specific ID to track it down, like the user ID from customers.
type Job struct {
	ID      string
	Request *http.Request
}

// The worker function processes the job and sends the result to the Results channel.
func (w HttpWorkerPool) worker() {
	// We iterate over the channels of incoming jobs and jobs limited by pool so we just run them concurrently.
	// Because we have different jobs and different results we need to split jobs in the pool into different types.
	go func() {
		// Use waitgroup from the pool to wait for all the jobs and close it when done.
		defer w.wg.Done()
		for job := range w.CustomerJobs {
			// We use the pre-configured HTTP client to execute the request.

			// Hotfix to avoid 429 Too Many requests error with simple time sleep.
			time.Sleep(time.Millisecond * 200)
			resp, err := w.client.Do(job.Request)
			if err != nil {
				w.errChan <- fmt.Errorf("error in the AddCustomer worker job %s %s", job.ID, err)
			}

			resp.Body.Close()
			w.CustomerResults <- processor.Result{JobID: string(job.ID), Status: resp.Status, Error: err}
		}
	}()

	go func() {
		// Use waitgroup from the pool to wait for all the jobs and close it when done.
		defer w.wg.Done()
		for job := range w.DealJobs {
			// We use the pre-configured HTTP client to execute the request.

			// Hotfix to avoid 429 Too Many requests error with simple time sleep.
			time.Sleep(time.Millisecond * 200)
			resp, err := w.client.Do(job.Request)
			if err != nil {
				w.errChan <- fmt.Errorf("error in the AddDeal worker job %s %s", job.ID, err)
			}

			resp.Body.Close()
			w.DealResults <- processor.Result{JobID: string(job.ID), Status: resp.Status, Error: err}
		}
	}()

	go func() {
		// Use waitgroup from the pool to wait for all the jobs and close it when done.
		defer w.wg.Done()
		for job := range w.PatchDealJobs {
			time.Sleep(time.Millisecond * 200)
			// We use the pre-configured HTTP client to execute the request.

			// Hotfix to avoid 429 Too Many requests error with simple time sleep.
			time.Sleep(time.Millisecond * 200)
			resp, err := w.client.Do(job.Request)
			if err != nil {
				w.errChan <- fmt.Errorf("error in the PatchDeal worker job %s %s", job.ID, err)
			} else if resp.StatusCode != http.StatusOK {
				w.errChan <- fmt.Errorf("error in the PatchDeal worker job %s Status: %s ", job.ID, resp.Status)
			}

			resp.Body.Close()
			w.PatchDealResults <- processor.Result{JobID: string(job.ID), Status: resp.Status, Error: err}
		}
	}()
}

func (w HttpWorkerPool) NewWorkerPool(ctx context.Context, errChan chan error, numWorkers int, client *http.Client, maxSize int) *HttpWorkerPool {
	var wg sync.WaitGroup

	pool := &HttpWorkerPool{
		Ctx:              ctx,
		client:           client,
		errChan:          errChan,
		CustomerJobs:     make(chan Job, maxSize),
		DealJobs:         make(chan Job, maxSize),
		PatchDealJobs:    make(chan Job, maxSize),
		DealResults:      make(chan processor.Result, maxSize),
		PatchDealResults: make(chan processor.Result, maxSize),
		CustomerResults:  make(chan processor.Result, maxSize),
		wg:               &wg,
	}

	pool.createWorkerPool(numWorkers)

	return pool
}

func (w HttpWorkerPool) createWorkerPool(numWorkers int) {
	for i := 1; i <= numWorkers; i++ {
		w.wg.Add(3) // We have 3 different types of job channels so we need more tokens.
		go w.worker()
	}
}

func (w HttpWorkerPool) addCustomerJob(job Job) {
	w.CustomerJobs <- job
}

func (w HttpWorkerPool) addDealJob(job Job) {
	w.DealJobs <- job
}

func (w HttpWorkerPool) addPatchDealJob(job Job) {
	w.PatchDealJobs <- job
}
