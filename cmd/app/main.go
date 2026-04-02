package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger.Info("Accounter — Starting Foundation...")
	os.Exit(0)
}
