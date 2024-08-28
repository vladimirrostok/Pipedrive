package daemon

import (
	"context"
	"pipedrive-sync/pipedrive-sync/internal/downloader"
	"pipedrive-sync/pipedrive-sync/internal/processor"
	"pipedrive-sync/pipedrive-sync/internal/reader"
	"pipedrive-sync/pipedrive-sync/internal/utils"
	"sync"
	"time"

	"go.uber.org/zap"
)

func RunDaemon(ctx context.Context, dataDownloader downloader.Downloader, dataReader reader.DataReader, dataProcessor processor.DataProcessor, client utils.HttpClient,
	downloadURL, outputFile []string, combinedDealsFile, combinedPaymentsFile string, interval int, errChan chan error) {
	// Create a WaitGroup to keep track of running goroutines.
	var wg sync.WaitGroup

	// When all the data is ready start the data processing loop, aka daemon process.
	wg.Add(1)
	go func() {
		zap.S().Infof("Starting the repetitive data processor service, aka daemon.")

		// Run the data processor periodically like a daemon as alternative to cron job.
		runRepetitiveSyncRunner(ctx, downloadURL, outputFile, combinedDealsFile, combinedPaymentsFile, dataProcessor, dataDownloader, dataReader, client, time.Duration(interval)*time.Second, errChan)
		wg.Done()
	}()

	// Wait for the services to finish.
	zap.S().Info("Waiting for the goroutines to finish.")
	wg.Wait()
}

// This is a function that runs task in the background and repeats it every passed duration.
func runRepetitiveSyncRunner(
	ctx context.Context, downloadURL, outputFile []string, combinedDealsFile, combinedPaymentsFile string,
	processor processor.DataProcessor, dataDownloader downloader.Downloader, dataReader reader.DataReader, client utils.HttpClient,
	d time.Duration, errChan chan error) {

	zap.S().Info("Starting the repetitive task runner.")

	// Create a ticker that will repeat the task every passed duration.
	ticker := time.NewTicker(d)
	defer func() {
		ticker.Stop()
		/*
			TODO: NB! Implement a graceful shutdown of the repetitive task on a deeper level as now it
			just stops the ticker and also shuts the app down without waiting for when the data processing ends.
		*/
	}()

	// We block the task from running multiple times at the same time.
	// Only one goroutine is allowed to run the sync task at a time.
	mu := sync.Mutex{}

	// Run the task in a loop until the context is closed or the ticker works.
	for {
		// Run the task on first go immediately after we initialized the ticker.
		go func() {
			mu.Lock()
			utils.SyncData(ctx, downloadURL, outputFile, combinedDealsFile, combinedPaymentsFile, processor, dataDownloader, dataReader, client, errChan)
			mu.Unlock()
		}()

		// Wait for the ticker to tick or the context to close.
		select {
		case <-ctx.Done():
			// TODO: Add a graceful shutdown of the repetitive task.
			// For now it just shuts down the ticker, context and HTTP timeouts should do the rest.
			zap.S().Info("Repetitive task stopped, finishing jobs.")
			return
		case <-ticker.C:
			// Check if the task is already running and processing the previous sync.
			start := mu.TryLock()
			if !start {
				// Warn the user that the previous task is still running.
				zap.S().Warn("The previous task is still running, skipping the current run.")
			} else {
				// Run the task if it's not running already.
				utils.SyncData(ctx, downloadURL, outputFile, combinedDealsFile, combinedPaymentsFile, processor, dataDownloader, dataReader, client, errChan)
				mu.Unlock()
			}
		}
	}
}
