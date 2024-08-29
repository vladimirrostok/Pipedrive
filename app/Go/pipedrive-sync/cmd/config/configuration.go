package config

// Config declares connection and service details.
type Config struct {
	DownloadBufferSize int `mapstructure:"download_buffer_size"`

	MaxReaderBufferSize int `mapstructure:"max_reader_buffer_size"`

	MaxDataProcessWorkers    int `mapstructure:"max_data_process_workers"`
	MaxHTTTPClentConnections int `mapstructure:"max_http_client_connections"`
	MaxHTTPClientTimeout     int `mapstructure:"max_http_client_timeout"`
	MaxPipelineSize          int `mapstructure:"max_pipeline_size"`

	RepetitiveTaskIntervalSec int `mapstructure:"repetitive_task_interval_sec"`

	ErrorsLogFile string `mapstructure:"errors_log_file.txt"`

	MaxProcessors int `mapstructure:"max_processors"`

	DownloadURL []string `mapstructure:"download_url"`
	OutputFile  []string `mapstructure:"output_file"`

	CombinedDealsFile    string `mapstructure:"combined_deals"`
	CombinedPaymentsFile string `mapstructure:"combined_payments"`

	CustomCustomerIDFieldKey string `mapstructure:"custom_customer_id_field_key"`
	CustomOrderIdFieldKey    string `mapstructure:"custom_order_id_field_key"`

	APIKey string // ENV variable
}
