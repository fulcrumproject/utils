package gormpg

import (
	"log/slog"
)

// Fulcrum Conf configuration
type Conf struct {
	DSN       string     `json:"dsn" env:"DSN" validate:"required"`
	LogLevel  slog.Level `json:"logLevel" env:"LOG_LEVEL"`
	LogFormat string     `json:"logFormat" env:"LOG_FORMAT" validate:"omitempty,oneof=text json"`
}
