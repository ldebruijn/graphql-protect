package accesslogging

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectProjectID(t *testing.T) {
	tests := []struct {
		name          string
		configValue   string
		envVars       map[string]string
		expectedID    string
		expectedError bool
	}{
		{
			name:          "uses config value when provided",
			configValue:   "config-project",
			envVars:       map[string]string{},
			expectedID:    "config-project",
			expectedError: false,
		},
		{
			name:        "uses GOOGLE_CLOUD_PROJECT env var",
			configValue: "",
			envVars: map[string]string{
				"GOOGLE_CLOUD_PROJECT": "env-project-1",
			},
			expectedID:    "env-project-1",
			expectedError: false,
		},
		{
			name:        "uses GCP_PROJECT env var when GOOGLE_CLOUD_PROJECT not set",
			configValue: "",
			envVars: map[string]string{
				"GCP_PROJECT": "env-project-2",
			},
			expectedID:    "env-project-2",
			expectedError: false,
		},
		{
			name:        "uses GCLOUD_PROJECT env var when others not set",
			configValue: "",
			envVars: map[string]string{
				"GCLOUD_PROJECT": "env-project-3",
			},
			expectedID:    "env-project-3",
			expectedError: false,
		},
		{
			name:        "prioritizes GOOGLE_CLOUD_PROJECT over others",
			configValue: "",
			envVars: map[string]string{
				"GOOGLE_CLOUD_PROJECT": "env-project-1",
				"GCP_PROJECT":          "env-project-2",
				"GCLOUD_PROJECT":       "env-project-3",
			},
			expectedID:    "env-project-1",
			expectedError: false,
		},
		{
			name:          "returns error when no project ID found",
			configValue:   "",
			envVars:       map[string]string{},
			expectedID:    "",
			expectedError: true,
		},
		{
			name:        "config value takes precedence over env vars",
			configValue: "config-project",
			envVars: map[string]string{
				"GOOGLE_CLOUD_PROJECT": "env-project",
			},
			expectedID:    "config-project",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars first
			os.Unsetenv("GOOGLE_CLOUD_PROJECT")
			os.Unsetenv("GCP_PROJECT")
			os.Unsetenv("GCLOUD_PROJECT")

			// Set test env vars
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				// Clean up after test
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			projectID, err := detectProjectID(tt.configValue)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "GCP project ID not found")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, projectID)
			}
		})
	}
}
