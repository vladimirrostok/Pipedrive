package http_buff_downloader

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
)

type HTTPBuffDownloader struct {
	Client     *http.Client
	BufferSize int
}

// The downloader function reads file over HTTP GET request and writes data to the disk simultaneously.
// NB! This processes data in batches, As it reads and writes simultaneously, if you stop it unfinished you will have a partial file.
func (h HTTPBuffDownloader) Download(ctx context.Context, path, output string) error {
	if path == "" || output == "" {
		return errors.New("check file path and name, these cannot be blank")
	}

	// Let's measure the time it takes to process the file.
	start := time.Now()
	zap.S().Infof("Creating the output data file: %s", output)

	// Create a file to write the data to.
	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()

	// Create a new HTTP request with the context, so we can cancel all the ongoing requests if needed.
	// When parent catches termination signal, it will cancel the context and all requests with context will be terminated.
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return err
	}

	zap.S().Infof("Downloading the: %s file from: %s", output, path)

	// The http.Response's Body has a Reader/Writer, that enables processing it in little batches in realtime.
	// We use the pre-configured HTTP client to execute the request.
	resp, err := h.Client.Do(req)
	if err != nil {
		if os.IsTimeout(err) {
			return errors.New("Timeout has been exceeded for file: " + output + " from: " + path + " with error: " + err.Error())
		}
		return errors.New("Downloading failed for file: " + output + " from: " + path + " with error: " + err.Error())
	}

	// Return the time it took to process the file and also close the response body.
	defer func() {
		resp.Body.Close()
		zap.S().Infof("Processing time for file %s : %s", output, time.Since(start))
	}()

	// io.Copy reads 32kb (maximum) buffered batches from input and writes them to output in realtime, then repeats.
	// Under the hood it uses a buffer to read and write data in batches sized 32 * 1024 bytes if no default buff provided.
	// We can also provide a custom buffer to io.Copy to optimize the performance with io.CopyBuffer().
	_, err = io.CopyBuffer(out, resp.Body, make([]byte, h.BufferSize))
	if err != nil {
		return err
	}

	zap.S().Infof("Data has been saved to file: %s", output)
	return nil
}
