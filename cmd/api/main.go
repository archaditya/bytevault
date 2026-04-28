package main

import (
	"os"

	"github.com/adityakkpk/bytevault/internal/config"
	"github.com/adityakkpk/bytevault/internal/logger"
	"github.com/adityakkpk/bytevault/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		os.Stdout.WriteString("Fail to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	logger.Init(cfg.App.Env)
	logger.Log.Info().Msg("Configuration Loaded")

	srv := server.New(cfg)
	if err:= srv.Start(); err != nil {
		logger.Log.Fatal().Err(err).Msg("❌ Server failed")
	}
}

