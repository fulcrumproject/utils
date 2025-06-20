package logging

import "log/slog"

// Fulcrum Conf configuration
type Conf struct {
	Format string     `json:"format" env:"LOG_FORMAT" validate:"omitempty,oneof=text json"`
	Level  slog.Level `json:"level" env:"LOG_LEVEL"`
}
