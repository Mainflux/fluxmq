package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mainflux/fluxmq/examples/simple"
	broker "github.com/mainflux/fluxmq/pkg/server"
	"github.com/mainflux/fluxmq/pkg/session"
	"github.com/mainflux/fluxmq/pkg/websocket"
	"go.uber.org/zap"
)

const (
	// HTTP
	defHTTPHost       = "0.0.0.0"
	defHTTPPort       = "8080"
	defHTTPScheme     = "ws"
	defHTTPTargetPath = "/mqtt"

	envHTTPHost       = "FLUXMQ_HTTP_HOST"
	envHTTPPort       = "FLUXMQ_HTTP_PORT"
	envHTTPScheme     = "FLUXMQ_HTTP_SCHEMA"
	envHTTPTargetPath = "FLUXMQ_HTTP_TARGET_PATH"

	// MQTT
	defMQTTHost = "0.0.0.0"
	defMQTTPort = "1883"

	envMQTTHost = "FLUXMQ_MQTT_HOST"
	envMQTTPort = "FLUXMQ_MQTT_PORT"

	defLogLevel = "debug"
	envLogLevel = "FLUXMQ_LOG_LEVEL"
)

type config struct {
	httpHost       string
	httpPort       string
	httpScheme     string
	httpTargetHost string
	httpTargetPort string
	httpTargetPath string

	mqttHost       string
	mqttPort       string
	mqttTargetHost string
	mqttTargetPort string

	logLevel string
}

func main() {
	cfg := loadConfig()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer logger.Sync()

	h := simple.New(logger)

	errs := make(chan error, 3)

	// HTTP
	logger.Info("Starting HTTP server on port " + cfg.httpPort)
	go serveHTTP(cfg, logger, h, errs)

	// MQTT
	logger.Info("Starting MQTT server on port " + cfg.mqttPort)
	go serveMQTT(cfg, logger, h, errs)

	go func() {
		c := make(chan os.Signal, 2)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Warn("FluxMQ terminated: " + err.Error())
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func loadConfig() config {
	return config{
		// HTTP
		httpHost:       env(envHTTPHost, defHTTPHost),
		httpPort:       env(envHTTPPort, defHTTPPort),
		httpScheme:     env(envHTTPScheme, defHTTPScheme),
		httpTargetPath: env(envHTTPTargetPath, defHTTPTargetPath),

		// MQTT
		mqttHost: env(envMQTTHost, defMQTTHost),
		mqttPort: env(envMQTTPort, defMQTTPort),

		// Log
		logLevel: env(envLogLevel, defLogLevel),
	}
}

func serveHTTP(cfg config, logger *zap.Logger, handler session.Handler, errs chan error) {
	target := fmt.Sprintf("%s:%s", cfg.httpTargetHost, cfg.httpTargetPort)
	wp := websocket.New(target, cfg.httpTargetPath, cfg.httpScheme, handler, logger)
	http.Handle("/mqtt", wp.Handler())

	p := fmt.Sprintf(":%s", cfg.httpPort)
	errs <- http.ListenAndServe(p, nil)
}

func serveMQTT(cfg config, logger *zap.Logger, handler session.Handler, errs chan error) {
	address := fmt.Sprintf("%s:%s", cfg.mqttHost, cfg.mqttPort)
	target := fmt.Sprintf("%s:%s", cfg.mqttTargetHost, cfg.mqttTargetPort)
	mqtt := broker.New(address, target, handler, logger)

	errs <- mqtt.ListenAndServe()
}
