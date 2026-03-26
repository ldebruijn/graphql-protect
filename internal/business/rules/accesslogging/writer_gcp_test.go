package accesslogging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectProjectID(t *testing.T) {
	tests := []struct {
		name          string
		configValue   string
		expectedID    string
		expectedError bool
	}{
		{
			name:          "returns project ID when configured",
			configValue:   "my-gcp-project",
			expectedID:    "my-gcp-project",
			expectedError: false,
		},
		{
			name:          "returns error when project ID not configured",
			configValue:   "",
			expectedID:    "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectID, err := detectProjectID(tt.configValue)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "GCP project ID not configured")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, projectID)
			}
		})
	}
}
