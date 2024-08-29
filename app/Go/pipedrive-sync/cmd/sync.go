package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"pipedrive-sync/pipedrive-sync/cmd/config"
	"pipedrive-sync/pipedrive-sync/internal/daemon"
	"pipedrive-sync/pipedrive-sync/internal/downloader"
	"pipedrive-sync/pipedrive-sync/internal/downloader/http_buff_downloader"
	"pipedrive-sync/pipedrive-sync/internal/processor"
	"pipedrive-sync/pipedrive-sync/internal/processor/pipeline_processor"
	"pipedrive-sync/pipedrive-sync/internal/reader"
	"pipedrive-sync/pipedrive-sync/internal/reader/buff_batch_reader"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var cfg config.Config

// Initialize logger to replace the default one to have a better timed log.
func initLogger() error {
	// Initialize the logs encoder.
	encoder := zap.NewProductionEncoderConfig()
	encoder.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	encoder.EncodeDuration = zapcore.NanosDurationEncoder

	// Initialize the logger.
	logger, err := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Encoding:         "console",
		EncoderConfig:    encoder,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}.Build()
	if err != nil {
		return err
	}

	// Replace the default logger with zap logger.
	zap.ReplaceGlobals(logger)

	return nil
}

// Load all configuration details from local .yaml file.
func loadConfiguration() error {
	viper.AddConfigPath("./config")
	viper.SetConfigName("configuration")

	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	err = viper.Unmarshal(&cfg)
	if err != nil {
		return err
	}

	cfg.APIKey = os.Getenv("PIPEDRIVE_API_KEY")
	if cfg.APIKey == "" {
		return errors.New("API key is missing")
	}

	return nil
}

func initHTTPDataDownloader(client *http.Client) downloader.Downloader {
	return &http_buff_downloader.HTTPBuffDownloader{
		Client:     client,
		BufferSize: cfg.DownloadBufferSize,
	}
}

func initDataReader(errChan chan error) reader.DataReader {
	return buff_batch_reader.BuffBatchReader{
		ErrChan:             errChan,
		MaxReaderBufferSize: cfg.MaxReaderBufferSize,
	}
}

func initDataProcessor(ctx context.Context, errChan chan error, client *http.Client, apiKey string) processor.DataProcessor {

	pool := &pipeline_processor.HttpWorkerPool{}
	pool = pool.NewWorkerPool(ctx, errChan, cfg.MaxDataProcessWorkers, client, cfg.MaxPipelineSize)

	return pipeline_processor.PipelineProcessor{
		API_KEY:                  apiKey,
		CustomCustomerIDFieldKey: cfg.CustomCustomerIDFieldKey,
		CustomOrderIDFieldKey:    cfg.CustomOrderIdFieldKey,
		ErrChan:                  errChan,
		MaxSize:                  cfg.MaxPipelineSize,
		HttpWorkerPool:           *pool,
	}
}

func main() {
	// Create a new context, with its cancellation function.
	// We will use this context to signal the server and all functions to stop.
	ctx, cancel := context.WithCancel(context.Background())

	// Disable cert verification to use self-signed certificates for all-case use.
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Global logging synchronizer.
	// This ensures the logged data is flushed out of the buffer before program exits.
	defer func() {
		zap.S().Sync()
		// Always print a well-formatted message when the process ends and
		// logger will include the timestamp to the message.
		zap.S().Info("The process has exited.")
	}()

	err := initLogger()
	if err != nil {
		zap.S().Fatal(err)
	}

	err = loadConfiguration()
	if err != nil {
		zap.S().Fatal(err)
	}

	// Set the maximum number of processors that can be executing simultaneously.
	runtime.GOMAXPROCS(cfg.MaxProcessors)
	zap.S().Info("The number of processors set to: ", cfg.MaxProcessors)

	// Provide channel to catch errors from the services running in the background.
	errChan := make(chan error)

	// Create the pre-configured HTTP client with the HTTP request timeout.
	// Go handles pool of HTTP connections inside the client thus we can use single instance of it.
	httpClient := http.Client{
		Timeout:   time.Duration(cfg.DownloadTimeoutSec) * time.Second,
		Transport: &http.Transport{MaxConnsPerHost: cfg.MaxHTTTPClentConnections},
	}

	dataDownloader := initHTTPDataDownloader(&httpClient)
	dataReader := initDataReader(errChan)
	dataProcessor := initDataProcessor(ctx, errChan, &httpClient, cfg.APIKey)

	// Provide the buffered channel for OS process termination signals.
	signalChan := make(chan os.Signal, 1)
	// Listen to the OS termination signals.
	signal.Notify(
		signalChan,
		os.Interrupt,
		syscall.SIGHUP,  // kill -SIGHUP XXXX
		syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT XXXX
		syscall.SIGTERM, // kill -SIGTERM XXXX
	)

	zap.S().Info("Configuration loaded.")

	// Start listening to the channels before starting the service, and let services run in the background.
	go func() {
		// We wrap select into loop to handle more signals which may arrive wnen service is shutting down.
		for {
			// A select without a "default" blocks until one of its cases can run, then it executes that case.
			// It chooses one at random if multiple are ready and exits.
			select {
			// If there is some unexpected error, exit the service fast.
			// There is no recovery, we don't want to proceed with broken data and cause bigger problems.
			case err := <-errChan:
				zap.S().Fatal(err)
			// If there is a shutdown signal, let the service exit on its own.
			// Cancel the context to signal all nested services to stop and let app exit gracefully.
			case <-signalChan:
				cancel()
				zap.S().Info("The process is shutting down gracefully.")
			}
		}
	}()

	daemon.RunDaemon(
		ctx,
		dataDownloader, dataReader, dataProcessor, &httpClient,
		cfg.DownloadURL, cfg.OutputFile, cfg.CombinedDealsFile, cfg.CombinedPaymentsFile,
		cfg.RepetitiveTaskIntervalSec,
		errChan)
}
