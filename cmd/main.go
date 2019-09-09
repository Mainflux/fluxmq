package main

import (
	"github.com/mainflux/fluxmq/pkg/server"
	"go.uber.org/zap"
)

const (
	host string = "0.0.0.0"
	port int    = 1883
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting FluxMQ", zap.Int("port", port))

	// Create the server with appropriate options.
	s := server.New(host, port, logger)

	// Start things up. Block here until done.
	s.ListenAndServe()
}
