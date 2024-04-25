package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/maragudk/env"
	"github.com/maragudk/migrate"

	"canvas/storage"
	"canvas/util"
)

func main() {
	os.Exit(start())
}

func start() int {
	_ = env.Load()

	logEnv := env.GetStringOrDefault("LOG_ENV", "development")
	util.InitializeSlog(logEnv, "")

	if len(os.Args) < 2 {
		slog.Warn("Usage: migrate up|down|to")
		return 1
	}

	if os.Args[1] == "to" && len(os.Args) < 3 {
		slog.Info("Usage: migrate to <version>")
		return 1
	}

	db := storage.NewDatabase(storage.NewDatabaseOptions{
		Host:     env.GetStringOrDefault("DB_HOST", "localhost"),
		Port:     env.GetIntOrDefault("DB_PORT", 5432),
		User:     env.GetStringOrDefault("DB_USER", ""),
		Password: env.GetStringOrDefault("DB_PASSWORD", ""),
		Name:     env.GetStringOrDefault("DB_NAME", ""),
	})

	if err := db.Connect(); err != nil {
		slog.Error("Error connection to database", util.ErrAttr(err))
		return 1
	}

	fsys := os.DirFS("storage/migrations")
	var err error
	switch os.Args[1] {
	case "up":
		err = migrate.Up(context.Background(), db.DB.DB, fsys)
	case "down":
		err = migrate.Down(context.Background(), db.DB.DB, fsys)
	case "to":
		err = migrate.To(context.Background(), db.DB.DB, fsys, os.Args[2])
	default:
		slog.Error("Unknown command", slog.String("name", os.Args[1]))
		return 1
	}
	if err != nil {
		slog.Error("Error migrating", util.ErrAttr(err))
		return 1
	}

	slog.Info("Migration completed")
	return 0
}
