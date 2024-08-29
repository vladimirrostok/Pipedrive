package utils

import (
	"context"
	"math/rand"
	"pipedrive-sync/pipedrive-sync/internal/downloader"
	"pipedrive-sync/pipedrive-sync/internal/model"
	"pipedrive-sync/pipedrive-sync/internal/processor"
	"pipedrive-sync/pipedrive-sync/internal/reader"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

func SyncData(ctx context.Context, downloadUrl, outputFile []string, combinedDealsFile, combinedPaymentsFile string,
	processor processor.DataProcessor, dataDownloader downloader.Downloader, dataReader reader.DataReader, client HttpClient,
	timeout int,
	errChan chan error) {

	zap.S().Info("The sync task is starting.")

	var wg sync.WaitGroup

	/* ******************************* */
	// Step 1: Download the data.
	/* ******************************* */
	downloadFiles(downloadUrl, outputFile, dataDownloader, ctx, errChan, &wg)
	wg.Wait()
	zap.S().Info("Sync data downloaded.")

	/* ******************************* */
	// Step 2: Sum up payments and
	// concatenate the data for processing.
	/* ******************************* */

	err := SumPayments(dataReader, outputFile[2], combinedPaymentsFile, &wg, errChan)
	if err != nil {
		zap.S().Errorf("Error while summing up payments in the data: %v", err)
		errChan <- err
	}
	wg.Wait()

	// Logically connect the data and store it as a file.
	err = ConcatenateData(errChan, dataReader, outputFile, combinedDealsFile, combinedPaymentsFile)
	if err != nil {
		zap.S().Errorf("Error while merging the data: %v", err)
		errChan <- err
	}

	zap.S().Info("The data merging is done.")

	/* ******************************* */
	// Step 3: Make sure the data has changed since the last sync and we need to update it.
	// Run the parallel processing for the customers.
	/* ******************************* */

	syncPersons(outputFile, processor, dataReader, client, &wg, timeout, errChan)

	/* ******************************* */
	// Step 4: Sync the deals, we need to match raw data ID-s to real ID-s from Pipedrive.
	// Run the parallel processing for the deals.
	/* ******************************* */

	syncDeals(combinedDealsFile, processor, dataReader, client, &wg, timeout, errChan)

	// Run until all jobs are done.
	wg.Wait()
	zap.S().Info("The data processing daemon task just exited back to runner in 5s.")
	// Make additional extra pause before the next run for the better demo.
	time.Sleep(time.Second * 5)
}

func downloadFiles(downloadUrl, outputFile []string, dataDownloader downloader.Downloader, ctx context.Context, errChan chan error, wg *sync.WaitGroup) {
	// We assume the configuration is correct and all download paths have corresponding file names like <path,filename>.
	// First we download all the data files.
	for i := 0; i < len(downloadUrl); i++ {
		wg.Add(1)
		go func() {
			err := dataDownloader.Download(ctx, downloadUrl[i], outputFile[i])
			if err != nil {
				// Let the main handler decide what to do with an error.
				errChan <- err
			}
			wg.Done()
		}()
	}
}

func syncPersons(outputFile []string, dataProcessor processor.DataProcessor, dataReader reader.DataReader, client HttpClient, wg *sync.WaitGroup, timeout int, errChan chan error) {
	// Reuse HTTP client for all the API calls with a longer timeout than we passed in main.
	allPersons := getAllPersons(client, errChan)

	// Prepare a buffer for the incoming data batches to not block the goroutine writing to it.
	newCustomers := make(chan model.Customer, 10)

	// Read the incoming customers data in batches.
	customers := dataReader.ReadCustomers(outputFile[0], 10)

	// Prepare a channel for the results from the processor in beforehand.
	// When we exit the function we close the channel to signal the end of the processing.
	// This will also stop the goroutine that's reading from the channel.
	results := make(chan processor.Result)
	done := make(chan bool)

	// Go through results we're receiving from the reader batch and data from Pipedrive.
	go func() {
		zap.S().Info("Processing customers to see if we need to sync any.")
		for custArr := range customers {

			/* ********************************************** */
			/* ********************************************** */
			/* FOR TEST ONLY */
			/* USE THIS TO SIMULATE LOAD */
			// Use this to simulate load, to show that all channels work concurrently.
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(50)))
			/* ********************************************** */
			/* ********************************************** */

			zap.S().Infof("Processing batch of: %d customers to see if we need sync any", len(custArr))
			for customer := range custArr {
				isFound := false
				// Loop over the persons array and find the corresponding customer ID.
				for persons := range allPersons.Data {
					// Verify local ID against real ID from Pipedrive, if these match we don't need to update.
					if custArr[customer].ID == allPersons.Data[persons].RealCustomerId {
						isFound = true
					}
					// If the customer is found, we don't need to add it to the new customers.
				}
				if !isFound {
					// If we didn't find this customer anywhere in the Pipedrive, add it to the new customers.
					newCustomers <- custArr[customer]
				}
			}
		}

		// Any reader still reading from buffered channel will receive item until the channel is empty.
		close(newCustomers)
		zap.S().Infof("Exiting the persons processor in %d seconds", +timeout*3)

		// When we read all the data, take x3 time from HTTP connection timeout
		// and use this as a time to exit the function, this should be enough to
		// process all the leftover data in the pool, if there is any.
		// If workers pool has same size as jobs pool size, by x1 timeout every
		// worker should quit already, x3 is to be extra sure.
		// NB! it's hotfix for now before task deadline.
		timeOut := time.After((time.Duration(timeout) * time.Second) * 3)
		// Select without default is a blocking operation.
		select {
		case <-timeOut:
			zap.S().Info("Exited the customer processing goroutine after x3 from HTTP timeout.")
			done <- true
			return
		}
	}()

	wg.Add(1)
	// Write new customers to the Pipedrive if some are out of sync.
	_, results = dataProcessor.ProcessCustomers(newCustomers)
	go func() {
		defer func() {
			wg.Done()
			zap.S().Info("Customer processing done, all jobs sent to worker pool if there were any.")
			zap.S().Info("Gracefully exited persons sync function.")
		}()

	loop:
		for {
			select {
			case result := <-results:
				{
					if result.Error != nil {
						zap.S().Warnf("Deal Job ID %s failed: %v ", result.JobID, result.Error)
					} else {
						zap.S().Infof("Deal Job ID %s succeeded: %v ", result.JobID, result.Status)
					}
				}
			case <-done:
				break loop
			}
		}
	}()
}

func syncDeals(combinedFile string, dataProcessor processor.DataProcessor, dataReader reader.DataReader, client HttpClient, wg *sync.WaitGroup, timeout int, errChan chan error) {
	// Reuse HTTP client for all the API calls with a longer timeout than we passed in main.
	allDeals := getAllDeals(client, errChan)

	// Prepare a buffer for the incoming data batches to not block the goroutine writing to it.
	newDeals := make(chan model.Deal, 10)
	outOfSyncDeals := make(chan model.Deal, 10)

	// Read the incoming deals data in batches.
	deals := dataReader.ReadDeals(combinedFile, 10)

	// Prepare a channel for the results from the processor in beforehand.
	// When we exit the function we close the channel to signal the end of the processing.
	// This will also stop the goroutine that's reading from the channel.
	results := make(chan processor.Result)
	outOfSyncResults := make(chan processor.Result)
	resultsDone := make(chan bool)
	outOfSyncResultsDone := make(chan bool)

	go func() {
		zap.S().Info("Processing deals to see if we need to sync any.")
		for dealArr := range deals {
			zap.S().Infof("Processing batch of: %d deals to see if we need sync any", len(dealArr))
			for deal := range dealArr {
				isFound := false
				for apiDeal := range allDeals.Data {
					// Loop over the deals array and find the corresponding deal.
					if dealArr[deal].OrderID == allDeals.Data[apiDeal].RealOrderId {
						// Update the local deal with the real ID from Pipedrive.
						dealArr[deal].OrderID = strconv.Itoa(allDeals.Data[apiDeal].ID)

						// Mark this deal as found, we don't need to add it to the new deals.
						isFound = true

						/* ********************************************** */
						/* ********************************************** */
						/* FOR TEST ONLY */
						/* SIMULATE DATA CHANGES IN THE INCOMING CSV FILE */
						// Simulate the probability of data changes with 5% chance out of 100%.
						if (rand.Intn(99)) >= 94 {
							// Simulate the data change, update value to trigger the update.
							dealArr[deal].OrderValue = strconv.Itoa(allDeals.Data[apiDeal].Value * 2)
						}
						/* ********************************************** */
						/* ********************************************** */

						// If values does't mach, add job to sync the deal.
						if dealArr[deal].OrderValue != strconv.Itoa(allDeals.Data[apiDeal].Value) {
							// Add job the out of sync deals processor.
							outOfSyncDeals <- dealArr[deal]
						}
						// If the deal is found, we don't need to add it to the new deals.
					}
				}
				// If we didn't find this deal anywhere in the Pipedrive, add it to the new deals.
				if !isFound {
					// Add job to create a new deal.
					newDeals <- dealArr[deal]
				}
			}
		}

		// Any reader still reading from buffered channel will receive item until the channel is empty.
		close(newDeals)
		close(outOfSyncDeals)
		zap.S().Infof("Exiting the deals processor in %d seconds", +timeout*3)

		// When we read all the data, take x3 time from HTTP connection timeout
		// and use this as a time to exit the function, this should be enough to
		// process all the leftover data in the pool, if there is any.
		// If workers pool has same size as jobs pool size, by x1 timeout every
		// worker should quit already, x3 is to be extra sure.
		// NB! it's hotfix for now before task deadline.
		timeOut := time.After((time.Duration(timeout) * time.Second) * 3)
		// Select without default is a blocking operation.
		select {
		case <-timeOut:
			zap.S().Info("Exited the deals processing goroutine after x3 from HTTP timeout.")
			resultsDone <- true
			outOfSyncResultsDone <- true
			return
		}

	}()

	wg.Add(1)
	// Write new deals to the Pipedrive if some are out of sync.
	_, results = dataProcessor.ProcessDeals(newDeals)
	go func() {
		defer func() {
			wg.Done()
			zap.S().Info("Deal processing done, all jobs sent to pool if there were any.")
			zap.S().Info("Gracefully exited deals sync function.")
		}()

	loop:
		for {
			select {
			case result := <-results:
				{
					if result.Error != nil {
						zap.S().Warnf("Deal Job ID %s failed: %v ", result.JobID, result.Error)
					} else {
						zap.S().Infof("Deal Job ID %s succeeded: %v ", result.JobID, result.Status)
					}
				}
			case <-resultsDone:
				break loop
			}
		}
	}()

	wg.Add(1)
	// Write new deals to the Pipedrive if some are out of sync.
	_, outOfSyncResults = dataProcessor.ProcessOutOfSyncDeals(outOfSyncDeals)
	go func() {
		defer func() {
			wg.Done()
			zap.S().Info("Deal updating done, all jobs sent to pool if there were any.")
			zap.S().Info("Gracefully exited out of date deals sync function.")
		}()
	loop:
		for {
			select {
			case result := <-outOfSyncResults:
				{
					if result.Error != nil {
						zap.S().Warnf("Deal Job ID %s failed: %v ", result.JobID, result.Error)
					} else {
						zap.S().Infof("Deal Job ID %s succeeded: %v ", result.JobID, result.Status)
					}
				}
			case <-outOfSyncResultsDone:
				break loop
			}
		}
	}()
}
