package logging

import (
	"fmt"
	"log/slog"
	"os"
)

type Config struct {
	Handler string
}

func Setup(cfg Config) error {
	options := slog.HandlerOptions{}

	var handler slog.Handler
	switch cfg.Handler {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, &options)
	case "json", "":
		handler = slog.NewJSONHandler(os.Stdout, &options)
	default:
		return fmt.Errorf("unsupported handler: '%s'", cfg.Handler)
	}

	slog.SetDefault(slog.New(handler))
	return nil
}
