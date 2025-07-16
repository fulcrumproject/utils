package confbuilder

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/copier"
	"github.com/joho/godotenv"
)

// Builder implements a generic builder pattern for creating configuration instances
type Builder[T any] struct {
	config    T
	envPrefix string
	envTag    string
	envFiles  []string
	filepath  *string
}

// New returns a Builder with the provided default configuration and options
func New[T any](defaultConf T) *Builder[T] {
	b := &Builder[T]{
		config:    defaultConf,
		envPrefix: "",    // Default prefix
		envTag:    "env", // Default tag
		envFiles:  []string{},
		filepath:  nil, // No file path by default
	}

	return b
}

// EnvPrefix sets the environment variable prefix
func (b *Builder[T]) EnvPrefix(prefix string) *Builder[T] {
	b.envPrefix = prefix
	return b
}

// EnvTag sets the struct tag name for environment variables
func (b *Builder[T]) EnvTag(tag string) *Builder[T] {
	b.envTag = tag
	return b
}

// EnvFiles sets the environment files to load
func (b *Builder[T]) EnvFiles(files ...string) *Builder[T] {
	b.envFiles = files
	return b
}

// LoadFile loads configuration from a JSON file
func (b *Builder[T]) File(filepath *string) *Builder[T] {
	b.filepath = filepath
	return b
}

// Build validates and returns the final configuration
func (b *Builder[T]) Build() (T, error) {
	var config T

	// Check if the source config is nil (for pointer types)
	sourceValue := reflect.ValueOf(b.config)
	if sourceValue.Kind() == reflect.Ptr && sourceValue.IsNil() {
		return config, fmt.Errorf("cannot load environment variables into nil pointer")
	}

	// Use reflection to determine if T is a pointer type
	configType := reflect.TypeOf(config)
	isPointer := configType.Kind() == reflect.Ptr

	// Allocate memory for pointer types
	if isPointer {
		configValue := reflect.New(configType.Elem())
		config = configValue.Interface().(T)
	}

	// Determine the target for operations that need a pointer
	var target any
	if isPointer {
		target = config
	} else {
		target = &config
	}

	// Clone the config to avoid modifying the original instance
	if err := copier.Copy(target, b.config); err != nil {
		return config, fmt.Errorf("failed to clone config: %w", err)
	}

	// Load from file if specified
	if b.filepath != nil && *b.filepath != "" {
		data, err := os.ReadFile(*b.filepath)
		if err != nil {
			return config, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := json.Unmarshal(data, target); err != nil {
			return config, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Load environment files
	if err := loadEnvFromAncestors(b.envFiles...); err != nil {
		return config, fmt.Errorf("failed to load environment variables: %w", err)
	}

	// Load environment variables into struct
	if err := loadEnvToStruct(target, b.envPrefix, b.envTag, ""); err != nil {
		return config, fmt.Errorf("failed to override configuration from environment: %w", err)
	}

	// Validate the configuration
	v := validator.New()
	if err := v.Struct(target); err != nil {
		return config, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadEnvToStruct loads environment variables into struct fields and nested structs based on tags
func loadEnvToStruct(target any, prefix, tag, parentEnvPath string) error {
	v := reflect.ValueOf(target)

	// Dereference all pointer levels to get to the actual value
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return fmt.Errorf("cannot load environment variables into nil pointer")
		}
		v = v.Elem()
	}

	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get env tag or skip if not present
		// Check if field is a struct that needs recursive processing
		if fieldValue.Kind() == reflect.Struct {
			// Skip time.Duration which is technically a struct but should be treated as primitive
			if field.Type != reflect.TypeOf(time.Duration(0)) {
				// Get the env tag for this struct field to build the path
				fieldEnvTag, hasEnvTag := field.Tag.Lookup(tag)
				var newEnvPath string
				if hasEnvTag && fieldEnvTag != "" {
					if parentEnvPath == "" {
						newEnvPath = fieldEnvTag
					} else {
						newEnvPath = parentEnvPath + "_" + fieldEnvTag
					}
				} else {
					newEnvPath = parentEnvPath
				}

				if err := loadEnvToStruct(fieldValue.Addr().Interface(), prefix, tag, newEnvPath); err != nil {
					return fmt.Errorf("error loading sub config field %s: %w", field.Name, err)
				}
			}
		}

		envVar, ok := field.Tag.Lookup(tag)
		if !ok || envVar == "" {
			continue
		}

		// Build the full environment variable name by concatenating parent path with current field env tag
		var fullEnvVar string
		if parentEnvPath == "" {
			fullEnvVar = envVar
		} else {
			fullEnvVar = parentEnvPath + "_" + envVar
		}

		// Get value from environment or skip if empty
		envValue := os.Getenv(prefix + fullEnvVar)
		if envValue == "" {
			continue
		}

		// Set field value based on type
		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(envValue)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Type == reflect.TypeOf(time.Duration(0)) {
				// Handle time.Duration
				duration, err := time.ParseDuration(envValue)
				if err != nil {
					return fmt.Errorf("invalid duration value for %s: %w", envVar, err)
				}
				fieldValue.SetInt(int64(duration))
			} else if field.Type == reflect.TypeOf(slog.Level(0)) {
				// Handle slog.Level - support both numeric and string values
				var level slog.Level
				if err := level.UnmarshalText([]byte(strings.ToUpper(envValue))); err != nil {
					return fmt.Errorf("invalid slog level value for %s: %s", envVar, envValue)
				}
				fieldValue.SetInt(int64(level))
			} else {
				// Handle regular integers
				val, err := strconv.ParseInt(envValue, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid integer value for %s: %w", envVar, err)
				}
				fieldValue.SetInt(val)
			}

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val, err := strconv.ParseUint(envValue, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid unsigned integer value for %s: %w", envVar, err)
			}
			fieldValue.SetUint(val)

		case reflect.Float32, reflect.Float64:
			val, err := strconv.ParseFloat(envValue, 64)
			if err != nil {
				return fmt.Errorf("invalid float value for %s: %w", envVar, err)
			}
			fieldValue.SetFloat(val)

		case reflect.Bool:
			val, err := strconv.ParseBool(envValue)
			if err != nil {
				return fmt.Errorf("invalid boolean value for %s: %w", envVar, err)
			}
			fieldValue.SetBool(val)

		case reflect.Slice:
			// Handle []string specifically. Add other slice types if needed.
			if fieldValue.Type().Elem().Kind() == reflect.String {
				parts := strings.Split(envValue, ",")
				// Trim spaces from each part
				for i, p := range parts {
					parts[i] = strings.TrimSpace(p)
				}
				fieldValue.Set(reflect.ValueOf(parts))
			}
		}
	}

	return nil
}

// loadEnvFromAncestors searches for .env files from the current directory up to the root
func loadEnvFromAncestors(filesToTry ...string) error {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Track if we found any env files
	found := false

	// Start from current directory and move up
	dir := currentDir
	for {
		for _, fileName := range filesToTry {
			envPath := filepath.Join(dir, fileName)
			if _, err := os.Stat(envPath); err == nil {
				// File exists, load it
				if err := godotenv.Load(envPath); err == nil {
					slog.Info("Loading .env file", "file", envPath)
					found = true
				}
			}
		}

		// Stop if we've reached the root directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			break // We've reached the root
		}
		dir = parentDir
	}

	if !found {
		slog.Info("No .env files found in ancestor directories")
	}

	return nil
}
