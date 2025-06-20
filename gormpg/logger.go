package gormpg

import (
	"log/slog"
	"os"

	slogGorm "github.com/orandin/slog-gorm"
	gormLogger "gorm.io/gorm/logger"
)

// NewGormLogger configures the logger based on the log format and level from config
func NewGormLogger(cfg *Conf) gormLogger.Interface {
	var handler slog.Handler

	// Configure the options with the log level
	opts := &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}

	// Configure the handler based on format
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slogGorm.New(
		slogGorm.WithHandler(handler),
		slogGorm.WithTraceAll(),
	)
}
