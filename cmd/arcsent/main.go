package main

import (
	"context"
	"flag"
	"os"

	"github.com/ipsix/arcsent/internal/config"
	"github.com/ipsix/arcsent/internal/daemon"
	"github.com/ipsix/arcsent/internal/logging"
)

func main() {
	configPath := flag.String("config", config.DefaultConfigPath, "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		_, _ = os.Stderr.WriteString("config error: " + err.Error() + "\n")
		os.Exit(1)
	}

	logger := logging.New(cfg.Daemon.LogFormat)
	logger.Info("arcsent starting", logging.Field{Key: "config", Value: cfg.Redacted()})

	runner := daemon.New(cfg, logger)
	if err := runner.Run(context.Background()); err != nil {
		logger.Error("daemon exited with error", logging.Field{Key: "error", Value: err.Error()})
		os.Exit(1)
	}
}
