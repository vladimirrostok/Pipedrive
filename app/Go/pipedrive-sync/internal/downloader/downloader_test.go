package downloader_test

import (
	"context"
	mock_downloader "pipedrive-sync/pipedrive-sync/internal/downloader/mocks"

	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Generic Downloader", func() {
	var (
		mockCtrl       *gomock.Controller
		ctx            context.Context
		mockDownloader *mock_downloader.MockDownloader
	)

	Describe("Setting up mocked HTTP client for test", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			ctx = context.Background()
			mockDownloader = mock_downloader.NewMockDownloader(mockCtrl)
		})

		When("Downloading is requested", func() {
			BeforeEach(func() {
			})
			It("Attempts to download file at the given URL with the given client ", func() {
				mockDownloader.EXPECT().Download(ctx, "http://test.com/", "test.txt")
			})
		})
	})
})
