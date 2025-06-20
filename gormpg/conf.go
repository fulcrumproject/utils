package gormpg

import (
	"log/slog"
)

// Fulcrum Conf configuration
type Conf struct {
	DSN       string     `json:"dsn" env:"DB_DSN" validate:"required"`
	LogLevel  slog.Level `json:"logLevel" env:"DB_LOG_LEVEL"`
	LogFormat string     `json:"logFormat" env:"DB_LOG_FORMAT" validate:"omitempty,oneof=text json"`
}
