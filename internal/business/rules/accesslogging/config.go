package accesslogging

// Config represents access logging configuration
type Config struct {
	Enabled              bool              `yaml:"enabled"`
	IncludedHeaders      []string          `yaml:"include_headers"`
	IncludeOperationName bool              `yaml:"include_operation_name"`
	IncludeVariables     bool              `yaml:"include_variables"`
	IncludePayload       bool              `yaml:"include_payload"`
	Async                bool              `yaml:"async"`
	BufferSize           int               `yaml:"buffer_size"`
	GoogleCloudLogging   GoogleCloudConfig `yaml:"google_cloud_logging"`
}

// GoogleCloudConfig represents Google Cloud Logging configuration
type GoogleCloudConfig struct {
	Enabled   bool   `yaml:"enabled"`
	ProjectID string `yaml:"project_id"`
	LogName   string `yaml:"log_name"`
}

// DefaultConfig returns default configuration for access logging
func DefaultConfig() Config {
	return Config{
		Enabled:              true,
		IncludedHeaders:      nil,
		IncludeOperationName: true,
		IncludeVariables:     true,
		IncludePayload:       false,
		Async:                false,
		BufferSize:           1000,
		GoogleCloudLogging: GoogleCloudConfig{
			Enabled:   false,
			ProjectID: "",
			LogName:   "",
		},
	}
}
