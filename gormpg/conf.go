package gormpg

import (
	"log/slog"
)

// Fulcrum DB configuration
type DB struct {
	DSN       string     `json:"dsn" env:"DB_DSN" validate:"required"`
	LogLevel  slog.Level `json:"logLevel" env:"DB_LOG_LEVEL"`
	LogFormat string     `json:"logFormat" env:"DB_LOG_FORMAT" validate:"omitempty,oneof=text json"`
}
