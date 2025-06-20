package gormpg

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormLogger "gorm.io/gorm/logger"
)

func TestNewGormLogger(t *testing.T) {
	tests := []struct {
		name      string
		jsonCfg   string
		expectNil bool
	}{
		{
			name: "json format with info level",
			jsonCfg: `{
				"dsn": "test-dsn",
				"logFormat": "json",
				"logLevel": "INFO"
			}`,
			expectNil: false,
		},
		{
			name: "text format with error level",
			jsonCfg: `{
				"dsn": "test-dsn",
				"logFormat": "text",
				"logLevel": "ERROR"
			}`,
			expectNil: false,
		},
		{
			name: "text format with warn level",
			jsonCfg: `{
				"dsn": "test-dsn",
				"logFormat": "text",
				"logLevel": "WARN"
			}`,
			expectNil: false,
		},
		{
			name: "text format with debug level",
			jsonCfg: `{
				"dsn": "test-dsn",
				"logFormat": "text",
				"logLevel": "DEBUG"
			}`,
			expectNil: false,
		},
		{
			name: "default format (text) with default level",
			jsonCfg: `{
				"dsn": "test-dsn",
				"logFormat": "",
				"logLevel": "INFO"
			}`,
			expectNil: false,
		},
		{
			name: "json format with default level",
			jsonCfg: `{
				"dsn": "test-dsn",
				"logFormat": "json",
				"logLevel": "INFO"
			}`,
			expectNil: false,
		},
		{
			name: "unknown format defaults to text",
			jsonCfg: `{
				"dsn": "test-dsn",
				"logFormat": "unknown",
				"logLevel": "INFO"
			}`,
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg DB
			err := json.Unmarshal([]byte(tt.jsonCfg), &cfg)
			require.NoError(t, err)

			logger := NewGormLogger(&cfg)

			if tt.expectNil {
				assert.Nil(t, logger)
			} else {
				require.NotNil(t, logger)
				assert.Implements(t, (*gormLogger.Interface)(nil), logger)
			}
		})
	}
}
