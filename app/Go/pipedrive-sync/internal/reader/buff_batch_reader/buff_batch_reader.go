package buff_batch_reader

// A general structure to pre-configure the buffered reader.
// We can play with the buffer size to optimize it's performance and memory costs.
type BuffBatchReader struct {
	ErrChan             chan error
	MaxReaderBufferSize int
}
