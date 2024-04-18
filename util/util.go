package util

import (
	"log/slog"
	"os"
)

// InitializeSlog
func InitializeSlog(env, release string) {
	// common attributes attached to every log
	slogAttr := []slog.Attr{
		slog.Group("environment", slog.String("release", release), slog.String("env", env)),
	}

	var logHandler slog.Handler
	if env == "development" {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}).
			WithAttrs(slogAttr)
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}).WithAttrs(slogAttr)
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)
}
