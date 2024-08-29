package utils_test

import (
	"context"
	mock_downloader "pipedrive-sync/pipedrive-sync/internal/downloader/mocks"
	"pipedrive-sync/pipedrive-sync/internal/model"
	"pipedrive-sync/pipedrive-sync/internal/processor"
	mock_processor "pipedrive-sync/pipedrive-sync/internal/processor/mocks"
	mock_reader "pipedrive-sync/pipedrive-sync/internal/reader/mocks"
	"pipedrive-sync/pipedrive-sync/internal/utils"
	mock_client "pipedrive-sync/pipedrive-sync/internal/utils/mocks"

	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

// TODO: Test is not finished yet.
var _ = Describe("Generic synchronizer", func() {
	var (
		mockCtrl             *gomock.Controller
		ctx                  context.Context
		mockDownloader       *mock_downloader.MockDownloader
		mockReader           *mock_reader.MockDataReader
		mockProcessor        *mock_processor.MockDataProcessor
		mockClient           *mock_client.MockHttpClient
		downloadURL          []string
		outputFile           []string
		combinedDealsFile    string
		combinedPaymentsFile string
		errChan              chan error
	)

	Describe("Setting up mocks for test", func() {
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			ctx = context.Background()
			mockDownloader = mock_downloader.NewMockDownloader(mockCtrl)
			mockReader = mock_reader.NewMockDataReader(mockCtrl)
			mockProcessor = mock_processor.NewMockDataProcessor(mockCtrl)
			mockClient = mock_client.NewMockHttpClient(mockCtrl)
			downloadURL = []string{"http://test.com/", "http://test.com/", "http://test.com/"}
			outputFile = []string{"raw_customers.csv", "raw_orders.csv", "raw_payments.csv"}
			combinedDealsFile = "deals.txt"
			combinedPaymentsFile = "payments.txt"
			errChan = make(chan error)
		})

		When("Syncdata is requested", func() {
			BeforeEach(func() {
				mockDownloader.EXPECT().Download(ctx, downloadURL[0], outputFile[0]).Return(nil)
				mockDownloader.EXPECT().Download(ctx, downloadURL[1], outputFile[1]).Return(nil)
				mockDownloader.EXPECT().Download(ctx, downloadURL[2], outputFile[2]).Return(nil)

				// Calls for the downloader's client to execute the request
				mockClient.EXPECT().Do(gomock.Any()).Return(nil, nil)
				mockClient.EXPECT().Do(gomock.Any()).Return(nil, nil)
				mockClient.EXPECT().Do(gomock.Any()).Return(nil, nil)

				cusChan := make(chan []model.Customer)
				orderChan := make(chan []model.Order)
				paymentChan := make(chan []model.Payment)

				// Read the files.
				mockReader.EXPECT().ReadCustomers(outputFile[0], gomock.Any()).Return(cusChan)
				mockReader.EXPECT().ReadOrders(outputFile[1], gomock.Any()).Return(orderChan)
				mockReader.EXPECT().ReadPayments(outputFile[2], gomock.Any()).Return(paymentChan)

				cusDone := make(chan bool)
				cusRes := make(chan processor.Result)

				orderDone := make(chan bool)
				orderRes := make(chan processor.Result)

				mockProcessor.EXPECT().ProcessCustomers(cusChan).Return(cusDone, cusRes)
				mockProcessor.EXPECT().ProcessDeals(orderChan).Return(nil, nil)

				mockProcessor.EXPECT().ProcessCustomers(gomock.Any()).Return(cusDone, cusRes)
				mockProcessor.EXPECT().ProcessOrders(gomock.Any()).Return(orderDone, orderRes)

			})

			It("Executes the entire flow of data synchronization process and stucks in loop", func() {
				// Downloads the file.
				go func() {
					defer GinkgoRecover()
					utils.SyncData(
						ctx, downloadURL, outputFile, combinedDealsFile, combinedPaymentsFile,
						mockProcessor, mockDownloader, mockReader, mockClient,
						10, errChan)
				}()
			}, 3)

			AfterEach(func() {
				// Clean up, remove the files created by this test.
			})
		})
	})
})
