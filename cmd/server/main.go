package main

import (
	"log/slog"
	"os"

	"github.com/maxim-shishkov/auth-service/application"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := application.Run(log); err != nil {
		log.Error("fatal startup error", "error", err)
		os.Exit(1)
	}
}
