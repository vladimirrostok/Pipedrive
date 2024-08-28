package buff_batch_reader

import (
	"bufio"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"pipedrive-sync/pipedrive-sync/internal/model"
)

func (d BuffBatchReader) ReadOrders(filePath string, batchSize int) (results chan []model.Order) {
	// Buffered channel to store the results.
	results = make(chan []model.Order, batchSize)

	if filePath == "" {
		d.ErrChan <- errors.New("check file path")
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		d.ErrChan <- err
		return
	}

	// Use buffered reader to read file in batches of size we need.
	buffReader := csv.NewReader(bufio.NewReaderSize(file, d.MaxReaderBufferSize))
	if _, err := buffReader.Read(); err != nil { // Read and skip the header line.
		d.ErrChan <- err
		return
	}

	// Run this in goroutine to return the channel asap, and fill it with data.
	go func() {
		// Always close the channel and the file.
		defer func() {
			close(results)
			file.Close()
		}()

		// We will read the file in batches, so we need to store the slice somewhere.
		// NB! batch has no pre-allocated capacity, so it will be reallocated on each append when it's over the cap.
		// TODO: optimize this by pre-allocating the capacity for the slice, probably make it as batch size
		// and make buffered channel two times larger for example, to have the buffer capacity.
		var batch []model.Order

		// Read the file line by line in batches.
		for {
			record, err := buffReader.Read()
			if err != nil {
				if err != io.EOF {
					d.ErrChan <- err
				}
				break
			}

			// Compose a customer from the record.
			order := model.Order{
				ID:        record[0],
				UserID:    record[1],
				OrderDate: record[2],
				Status:    record[3],
			}

			batch = append(batch, order)

			// Fill the slice with output to required size.
			if len(batch) >= batchSize {
				results <- batch
				batch = nil // Reset slice for the next batch.
			}
		}

		// If there are any leftover records left after the read, send them as well.
		if len(batch) > 0 {
			results <- batch
		}

	}()

	return results
}
