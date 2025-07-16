package confbuilder

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DatabaseConfig represents a nested configuration structure
type DatabaseConfig struct {
	Host     string `json:"host" env:"HOST" validate:"required,hostname_rfc1123"`
	Port     int    `json:"port" env:"PORT" validate:"required,min=1,max=65535"`
	Username string `json:"username" env:"USERNAME" validate:"required,min=3"`
	Password string `json:"password" env:"PASSWORD" validate:"required,min=8"`
	SSL      bool   `json:"ssl" env:"SSL"`
}

// ServerConfig represents another nested structure
type ServerConfig struct {
	Name    string        `json:"name" env:"NAME" validate:"required,alphanum,min=3,max=20"`
	Timeout time.Duration `json:"timeout" env:"TIMEOUT" validate:"required"`
	Workers uint          `json:"workers" env:"WORKERS" validate:"min=1,max=100"`
}

// TestConfig includes all supported field types, validation, and nested structs
type TestConfig struct {
	// Basic string types
	AppName     string `json:"app_name" env:"APP_NAME" validate:"required,min=3,max=50"`
	Environment string `json:"environment" env:"ENVIRONMENT" validate:"required,oneof=development staging production"`
	Version     string `json:"version" env:"VERSION" validate:"semver"`

	// Numeric types
	Port       int     `json:"port" env:"PORT" validate:"required,min=1,max=65535"`
	MaxConns   uint    `json:"max_conns" env:"MAX_CONNS" validate:"min=1,max=10000"`
	LoadFactor float64 `json:"load_factor" env:"LOAD_FACTOR" validate:"min=0.1,max=10.0"`
	Precision  float32 `json:"precision" env:"PRECISION" validate:"min=0.01,max=1.0"`

	// Boolean
	Enabled   bool `json:"enabled" env:"ENABLED"`
	DebugMode bool `json:"debug_mode" env:"DEBUG_MODE"`

	// Special types
	LogLevel    slog.Level    `json:"log_level" env:"LOG_LEVEL"`
	Timeout     time.Duration `json:"timeout" env:"TIMEOUT" validate:"required"`
	GracePeriod time.Duration `json:"grace_period" env:"GRACE_PERIOD"`

	// Slice types
	Tags       []string `json:"tags" env:"TAGS" validate:"required,min=1,dive,min=1"`
	AllowedIPs []string `json:"allowed_ips" env:"ALLOWED_IPS" validate:"dive,ip"`

	// Nested structs
	Database DatabaseConfig `json:"database" env:"DB"`
	Server   ServerConfig   `json:"server" env:"SERVER"`

	// Fields without env tags (should remain unchanged)
	InternalID string `json:"internal_id"`
	unexported string // Unexported field
}

// newTestConfig returns a default configuration for testing
func newTestConfig() *TestConfig {
	return &TestConfig{
		AppName:     "test-app",
		Environment: "development",
		Version:     "1.0.0",
		Port:        8080,
		MaxConns:    100,
		LoadFactor:  1.0,
		Precision:   0.5,
		Enabled:     true,
		DebugMode:   false,
		LogLevel:    slog.LevelInfo,
		Timeout:     30 * time.Second,
		GracePeriod: 5 * time.Second,
		Tags:        []string{"default"},
		AllowedIPs:  []string{"127.0.0.1"},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Username: "testuser",
			Password: "testpass123",
			SSL:      false,
		},
		Server: ServerConfig{
			Name:    "server01",
			Timeout: 10 * time.Second,
			Workers: 5,
		},
		InternalID: "internal-123",
		unexported: "unexported-value",
	}
}

// setEnvVars sets environment variables for testing and cleans them up
func setEnvVars(t *testing.T, envVars map[string]string) {
	t.Helper()
	for key, value := range envVars {
		os.Setenv(key, value)
	}
	t.Cleanup(func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
	})
}

func TestGenericBuilder_Build(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(t *testing.T) (*TestConfig, *Builder[*TestConfig])
		expectError  bool
		errorMsg     string
		validateFunc func(t *testing.T, cfg *TestConfig)
	}{
		{
			name: "default configuration",
			setupFunc: func(t *testing.T) (*TestConfig, *Builder[*TestConfig]) {
				defaultCfg := newTestConfig()
				builder := New(defaultCfg)
				return defaultCfg, builder
			},
			expectError: false,
			validateFunc: func(t *testing.T, cfg *TestConfig) {
				assert.Equal(t, "test-app", cfg.AppName)
				assert.Equal(t, "development", cfg.Environment)
				assert.Equal(t, 8080, cfg.Port)
				assert.Equal(t, uint(100), cfg.MaxConns)
				assert.True(t, cfg.Enabled)
				assert.Equal(t, slog.LevelInfo, cfg.LogLevel)
				assert.Equal(t, []string{"default"}, cfg.Tags)
				assert.Equal(t, "localhost", cfg.Database.Host)
				assert.Equal(t, "server01", cfg.Server.Name)
			},
		},
		{
			name: "environment variables override",
			setupFunc: func(t *testing.T) (*TestConfig, *Builder[*TestConfig]) {
				setEnvVars(t, map[string]string{
					"TEST_APP_NAME":       "env-app",
					"TEST_ENVIRONMENT":    "production",
					"TEST_PORT":           "9090",
					"TEST_MAX_CONNS":      "500",
					"TEST_LOAD_FACTOR":    "2.5",
					"TEST_PRECISION":      "0.75",
					"TEST_ENABLED":        "false",
					"TEST_DEBUG_MODE":     "true",
					"TEST_LOG_LEVEL":      "ERROR",
					"TEST_TIMEOUT":        "60s",
					"TEST_GRACE_PERIOD":   "10s",
					"TEST_TAGS":           "env,production,api",
					"TEST_ALLOWED_IPS":    "192.168.1.1,10.0.0.1",
					"TEST_DB_HOST":        "prod-db",
					"TEST_DB_PORT":        "3306",
					"TEST_DB_USERNAME":    "produser",
					"TEST_DB_PASSWORD":    "prodpass123",
					"TEST_DB_SSL":         "true",
					"TEST_SERVER_NAME":    "prodserver",
					"TEST_SERVER_TIMEOUT": "30s",
					"TEST_SERVER_WORKERS": "20",
				})
				defaultCfg := newTestConfig()
				builder := New(defaultCfg).EnvPrefix("TEST_").EnvTag("env")
				return defaultCfg, builder
			},
			expectError: false,
			validateFunc: func(t *testing.T, cfg *TestConfig) {
				assert.Equal(t, "env-app", cfg.AppName)
				assert.Equal(t, "production", cfg.Environment)
				assert.Equal(t, 9090, cfg.Port)
				assert.Equal(t, uint(500), cfg.MaxConns)
				assert.Equal(t, 2.5, cfg.LoadFactor)
				assert.Equal(t, float32(0.75), cfg.Precision)
				assert.False(t, cfg.Enabled)
				assert.True(t, cfg.DebugMode)
				assert.Equal(t, slog.LevelError, cfg.LogLevel)
				assert.Equal(t, 60*time.Second, cfg.Timeout)
				assert.Equal(t, 10*time.Second, cfg.GracePeriod)
				assert.Equal(t, []string{"env", "production", "api"}, cfg.Tags)
				assert.Equal(t, []string{"192.168.1.1", "10.0.0.1"}, cfg.AllowedIPs)
				assert.Equal(t, "prod-db", cfg.Database.Host)
				assert.Equal(t, 3306, cfg.Database.Port)
				assert.Equal(t, "produser", cfg.Database.Username)
				assert.Equal(t, "prodpass123", cfg.Database.Password)
				assert.True(t, cfg.Database.SSL)
				assert.Equal(t, "prodserver", cfg.Server.Name)
				assert.Equal(t, 30*time.Second, cfg.Server.Timeout)
				assert.Equal(t, uint(20), cfg.Server.Workers)
			},
		},
		{
			name: "nil pointer error",
			setupFunc: func(t *testing.T) (*TestConfig, *Builder[*TestConfig]) {
				builder := New((*TestConfig)(nil))
				return nil, builder
			},
			expectError: true,
			errorMsg:    "cannot load environment variables into nil pointer",
		},
		{
			name: "validation error - missing required field",
			setupFunc: func(t *testing.T) (*TestConfig, *Builder[*TestConfig]) {
				defaultCfg := newTestConfig()
				defaultCfg.AppName = "" // Required field
				builder := New(defaultCfg)
				return defaultCfg, builder
			},
			expectError: true,
			errorMsg:    "invalid configuration",
		},
		{
			name: "validation error - invalid environment",
			setupFunc: func(t *testing.T) (*TestConfig, *Builder[*TestConfig]) {
				defaultCfg := newTestConfig()
				defaultCfg.Environment = "invalid" // Must be one of: development, staging, production
				builder := New(defaultCfg)
				return defaultCfg, builder
			},
			expectError: true,
			errorMsg:    "invalid configuration",
		},
		{
			name: "validation error - port out of range",
			setupFunc: func(t *testing.T) (*TestConfig, *Builder[*TestConfig]) {
				defaultCfg := newTestConfig()
				defaultCfg.Port = 70000 // Exceeds max port
				builder := New(defaultCfg)
				return defaultCfg, builder
			},
			expectError: true,
			errorMsg:    "invalid configuration",
		},
		{
			name: "validation error - nested struct validation",
			setupFunc: func(t *testing.T) (*TestConfig, *Builder[*TestConfig]) {
				defaultCfg := newTestConfig()
				defaultCfg.Database.Password = "short" // Less than 8 characters
				builder := New(defaultCfg)
				return defaultCfg, builder
			},
			expectError: true,
			errorMsg:    "invalid configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, builder := tt.setupFunc(t)
			cfg, err := builder.Build()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				if tt.validateFunc != nil {
					tt.validateFunc(t, cfg)
				}
			}
		})
	}
}

func TestGenericBuilder_FileLoading(t *testing.T) {
	tests := []struct {
		name         string
		fileContent  string
		expectError  bool
		errorMsg     string
		validateFunc func(t *testing.T, cfg *TestConfig)
	}{
		{
			name: "valid JSON file",
			fileContent: `{
				"app_name": "file-app",
				"environment": "staging",
				"port": 9090,
				"max_conns": 200,
				"load_factor": 1.5,
				"enabled": false,
				"tags": ["file", "config"],
				"database": {
					"host": "file-db",
					"port": 3306,
					"username": "fileuser",
					"password": "filepass123"
				},
				"server": {
					"name": "fileserver",
					"timeout": 20000000000,
					"workers": 10
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg *TestConfig) {
				assert.Equal(t, "file-app", cfg.AppName)
				assert.Equal(t, "staging", cfg.Environment)
				assert.Equal(t, 9090, cfg.Port)
				assert.Equal(t, uint(200), cfg.MaxConns)
				assert.Equal(t, 1.5, cfg.LoadFactor)
				assert.False(t, cfg.Enabled)
				assert.Equal(t, []string{"file", "config"}, cfg.Tags)
				assert.Equal(t, "file-db", cfg.Database.Host)
				assert.Equal(t, 3306, cfg.Database.Port)
				assert.Equal(t, "fileserver", cfg.Server.Name)
				assert.Equal(t, uint(10), cfg.Server.Workers)
			},
		},
		{
			name:        "invalid JSON",
			fileContent: `{invalid json}`,
			expectError: true,
			errorMsg:    "failed to parse config file",
		},
		{
			name: "validation error from file",
			fileContent: `{
				"app_name": "ab",
				"environment": "development",
				"port": 8080
			}`,
			expectError: true,
			errorMsg:    "invalid configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.json")
			err := os.WriteFile(configPath, []byte(tt.fileContent), 0644)
			require.NoError(t, err)

			defaultCfg := newTestConfig()
			cfg, err := New(defaultCfg).File(&configPath).Build()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				if tt.validateFunc != nil {
					tt.validateFunc(t, cfg)
				}
			}
		})
	}

	t.Run("non-existent file", func(t *testing.T) {
		nonExistentFile := "/path/to/nonexistent/config.json"
		defaultCfg := newTestConfig()
		_, err := New(defaultCfg).File(&nonExistentFile).Build()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("nil and empty filepath", func(t *testing.T) {
		defaultCfg := newTestConfig()

		// Test nil filepath
		cfg, err := New(defaultCfg).File(nil).Build()
		require.NoError(t, err)
		assert.Equal(t, "test-app", cfg.AppName)

		// Test empty filepath
		emptyPath := ""
		cfg, err = New(defaultCfg).File(&emptyPath).Build()
		require.NoError(t, err)
		assert.Equal(t, "test-app", cfg.AppName)
	})
}

func TestGenericBuilder_EnvParsingErrors(t *testing.T) {
	tests := []struct {
		name        string
		envVarKey   string
		envVarValue string
		expectedErr string
	}{
		{
			name:        "invalid int",
			envVarKey:   "TEST_PORT",
			envVarValue: "not-an-int",
			expectedErr: "invalid syntax",
		},
		{
			name:        "invalid uint",
			envVarKey:   "TEST_MAX_CONNS",
			envVarValue: "-1",
			expectedErr: "invalid syntax",
		},
		{
			name:        "invalid float64",
			envVarKey:   "TEST_LOAD_FACTOR",
			envVarValue: "not-a-float",
			expectedErr: "invalid syntax",
		},
		{
			name:        "invalid float32",
			envVarKey:   "TEST_PRECISION",
			envVarValue: "not-a-float",
			expectedErr: "invalid syntax",
		},
		{
			name:        "invalid bool",
			envVarKey:   "TEST_ENABLED",
			envVarValue: "not-a-bool",
			expectedErr: "invalid syntax",
		},
		{
			name:        "invalid duration",
			envVarKey:   "TEST_TIMEOUT",
			envVarValue: "invalid-duration",
			expectedErr: "invalid duration value",
		},
		{
			name:        "invalid slog level",
			envVarKey:   "TEST_LOG_LEVEL",
			envVarValue: "INVALID_LEVEL",
			expectedErr: "invalid slog level value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvVars(t, map[string]string{tt.envVarKey: tt.envVarValue})

			defaultCfg := newTestConfig()
			_, err := New(defaultCfg).EnvPrefix("TEST_").EnvTag("env").Build()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestGenericBuilder_ChainedOperations(t *testing.T) {
	// Create a temporary file
	configJSON := `{
		"app_name": "file-app",
		"port": 9090,
		"load_factor": 2.0,
		"database": {
			"host": "file-db"
		}
	}`
	configPath := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	setEnvVars(t, map[string]string{
		"TEST_PORT":        "6060",
		"TEST_TAGS":        "env,tag",
		"TEST_DB_USERNAME": "envuser",
	})

	// Chain operations: default -> file -> env
	defaultCfg := newTestConfig()
	cfg, err := New(defaultCfg).
		EnvPrefix("TEST_").
		EnvTag("env").
		File(&configPath).
		Build()
	require.NoError(t, err)

	assert.Equal(t, "file-app", cfg.AppName)              // From file
	assert.Equal(t, 6060, cfg.Port)                       // From env (overrides file)
	assert.Equal(t, "development", cfg.Environment)       // From default
	assert.Equal(t, 2.0, cfg.LoadFactor)                  // From file
	assert.Equal(t, []string{"env", "tag"}, cfg.Tags)     // From env
	assert.Equal(t, "file-db", cfg.Database.Host)         // From file
	assert.Equal(t, "envuser", cfg.Database.Username)     // From env
	assert.Equal(t, "testpass123", cfg.Database.Password) // From default
}

func TestGenericBuilder_EnvFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create env files
	envFile1Content := "TEST_APP_NAME=name-from-file1\nTEST_PORT=1111"
	envFile1Path := filepath.Join(tempDir, ".env.test1")
	err := os.WriteFile(envFile1Path, []byte(envFile1Content), 0644)
	require.NoError(t, err)

	envFile2Content := "TEST_PORT=2222\nTEST_LOAD_FACTOR=7.7"
	envFile2Path := filepath.Join(tempDir, ".env.test2")
	err = os.WriteFile(envFile2Path, []byte(envFile2Content), 0644)
	require.NoError(t, err)

	// Set actual environment variables
	setEnvVars(t, map[string]string{
		"TEST_APP_NAME": "name-from-actual-env",
		"TEST_TAGS":     "actual,env",
	})

	// Change working directory to tempDir
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	t.Cleanup(func() { os.Chdir(originalWD) })

	defaultCfg := newTestConfig()
	cfg, err := New(defaultCfg).
		EnvPrefix("TEST_").
		EnvTag("env").
		EnvFiles(".env.test1", ".env.test2").
		Build()
	require.NoError(t, err)

	assert.Equal(t, "name-from-actual-env", cfg.AppName) // From actual env
	assert.Equal(t, 1111, cfg.Port)                      // From .env.test1 (first file wins)
	assert.Equal(t, "development", cfg.Environment)      // Default value
	assert.Equal(t, 7.7, cfg.LoadFactor)                 // From .env.test2
	assert.Equal(t, []string{"actual", "env"}, cfg.Tags) // From actual env
}

func TestGenericBuilder_SliceHandling(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected []string
	}{
		{
			name:     "single value",
			envValue: "single",
			expected: []string{"single"},
		},
		{
			name:     "multiple values",
			envValue: "tag1,tag2,tag3",
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "values with spaces",
			envValue: " tag1 , tag2 , tag3 ",
			expected: []string{"tag1", "tag2", "tag3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvVars(t, map[string]string{"TEST_TAGS": tt.envValue})

			defaultCfg := newTestConfig()
			cfg, err := New(defaultCfg).EnvPrefix("TEST_").EnvTag("env").Build()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Tags)
		})
	}
}

func TestGenericBuilder_ConfigCloning(t *testing.T) {
	// Test that the original config is not modified
	originalCfg := newTestConfig()
	originalAppName := originalCfg.AppName
	originalPort := originalCfg.Port
	originalDBHost := originalCfg.Database.Host

	setEnvVars(t, map[string]string{
		"TEST_APP_NAME": "modified-name",
		"TEST_PORT":     "9999",
		"TEST_DB_HOST":  "modified-db",
	})

	cfg, err := New(originalCfg).EnvPrefix("TEST_").EnvTag("env").Build()
	require.NoError(t, err)

	// Check that the returned config has the modified values
	assert.Equal(t, "modified-name", cfg.AppName)
	assert.Equal(t, 9999, cfg.Port)
	assert.Equal(t, "modified-db", cfg.Database.Host)

	// Check that the original config is unchanged
	assert.Equal(t, originalAppName, originalCfg.AppName)
	assert.Equal(t, originalPort, originalCfg.Port)
	assert.Equal(t, originalDBHost, originalCfg.Database.Host)
}

func TestGenericBuilder_EmptyEnvValues(t *testing.T) {
	setEnvVars(t, map[string]string{
		"TEST_APP_NAME":    "",
		"TEST_LOAD_FACTOR": "",
		"TEST_PORT":        "7070",
	})

	defaultCfg := newTestConfig()
	cfg, err := New(defaultCfg).EnvPrefix("TEST_").EnvTag("env").Build()
	require.NoError(t, err)

	// Should keep default values for empty env vars
	assert.Equal(t, "test-app", cfg.AppName)
	assert.Equal(t, 1.0, cfg.LoadFactor)
	// Port should be updated
	assert.Equal(t, 7070, cfg.Port)
}

func TestGenericBuilder_FieldsWithoutEnvTags(t *testing.T) {
	setEnvVars(t, map[string]string{
		"TEST_APP_NAME":    "new-name",
		"TEST_INTERNAL_ID": "should-be-ignored", // No env tag
	})

	defaultCfg := newTestConfig()
	cfg, err := New(defaultCfg).EnvPrefix("TEST_").EnvTag("env").Build()
	require.NoError(t, err)

	assert.Equal(t, "new-name", cfg.AppName)
	assert.Equal(t, "internal-123", cfg.InternalID)     // Should remain unchanged
	assert.Equal(t, "unexported-value", cfg.unexported) // Unexported fields preserved
}

func TestGenericBuilder_ConcatenatedEnvTags(t *testing.T) {
	t.Run("nested struct env tag concatenation", func(t *testing.T) {
		setEnvVars(t, map[string]string{
			// These should work with concatenated env tags:
			// Database struct has env:"DB", Database.Host has env:"HOST" -> "TEST_DB_HOST"
			// Server struct has env:"SERVER", Server.Name has env:"NAME" -> "TEST_SERVER_NAME"
			"TEST_DB_HOST":        "concat-db-host",
			"TEST_DB_PORT":        "9999",
			"TEST_DB_USERNAME":    "concat-user",
			"TEST_SERVER_NAME":    "concatserver",
			"TEST_SERVER_TIMEOUT": "45s",
			"TEST_SERVER_WORKERS": "25",
		})

		defaultCfg := newTestConfig()
		cfg, err := New(defaultCfg).EnvPrefix("TEST_").EnvTag("env").Build()
		require.NoError(t, err)

		// Verify database config values were set from concatenated env vars
		assert.Equal(t, "concat-db-host", cfg.Database.Host)
		assert.Equal(t, 9999, cfg.Database.Port)
		assert.Equal(t, "concat-user", cfg.Database.Username)
		assert.Equal(t, "testpass123", cfg.Database.Password) // Should remain default since TEST_DB_PASSWORD not set

		// Verify server config values were set from concatenated env vars
		assert.Equal(t, "concatserver", cfg.Server.Name)
		assert.Equal(t, 45*time.Second, cfg.Server.Timeout)
		assert.Equal(t, uint(25), cfg.Server.Workers)
	})

	t.Run("parent struct without env tag", func(t *testing.T) {
		// Test case where parent struct field doesn't have env tag
		type NestedConfig struct {
			Value string `env:"VALUE"`
		}
		type ParentConfig struct {
			AppName string       `env:"APP_NAME"`
			Nested  NestedConfig // No env tag on parent
		}

		setEnvVars(t, map[string]string{
			"TEST_APP_NAME": "parent-app",
			"TEST_VALUE":    "nested-value", // Should use just the field's env tag
		})

		defaultCfg := &ParentConfig{
			AppName: "default-app",
			Nested:  NestedConfig{Value: "default-value"},
		}

		cfg, err := New(defaultCfg).EnvPrefix("TEST_").EnvTag("env").Build()
		require.NoError(t, err)

		assert.Equal(t, "parent-app", cfg.AppName)
		assert.Equal(t, "nested-value", cfg.Nested.Value)
	})
}
