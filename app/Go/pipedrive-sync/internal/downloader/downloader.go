package downloader

import "context"

// Let the Downloader interface be implemented by any type that has a Download method and returns an error.
// This way we can pass any custom logic to download the file and signal on it's result.
type Downloader interface {
	Download(ctx context.Context, path, output string) error
}
