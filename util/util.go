package util

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"reflect"

	"github.com/caarlos0/env/v11"
	dotenv "github.com/maragudk/env"
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

// ErrAttr return slog.Attr of Any
// The key is error, and is the err
func ErrAttr(err error) slog.Attr {
	return slog.Any("error", err)
}

// custom parser for parsing string into net.Addr
// compatible with env.ParserFunc
func parseURL(value string) (any, error) {
	addr, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}
	return *addr, nil
}

// Builds config - error handling omitted fore brevity
func LoadConfig[Config any](c *Config) error {
	// Loading the environment variables from '.env' file.
	// ignore the error because on the server we will use the env variables from the OS Environment
	_ = dotenv.Load()

	return env.ParseWithOptions(
		c,
		env.Options{RequiredIfNoDef: true, FuncMap: map[reflect.Type]env.ParserFunc{
			reflect.TypeOf(url.URL{}): parseURL,
		}},
	)
}
