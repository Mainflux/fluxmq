package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mainflux/fluxmq/auth"
	"github.com/mainflux/fluxmq/client"
	"github.com/mainflux/fluxmq/examples/simple"
	broker "github.com/mainflux/fluxmq/server"
	"github.com/mainflux/fluxmq/session"
	"go.uber.org/zap"
)

const (
	// MQTT
	defMQTTHost = "0.0.0.0"
	defMQTTPort = "1883"

	envMQTTHost = "FLUXMQ_MQTT_HOST"
	envMQTTPort = "FLUXMQ_MQTT_PORT"

	defLogLevel = "debug"
	envLogLevel = "FLUXMQ_LOG_LEVEL"
)

type config struct {
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
	sRepo := session.NewRepository()
	cRepo := client.NewRepository()

	errs := make(chan error, 3)

	// MQTT
	logger.Info("Starting MQTT server on port " + cfg.mqttPort)
	go serveMQTT(cfg, logger, h, sRepo, cRepo, errs)

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
		// MQTT
		mqttHost: env(envMQTTHost, defMQTTHost),
		mqttPort: env(envMQTTPort, defMQTTPort),

		// Log
		logLevel: env(envLogLevel, defLogLevel),
	}
}

func serveMQTT(cfg config, logger *zap.Logger, handler auth.Handler,
	sRepo *session.Repository, cRepo *client.Repository, errs chan error) {
	address := fmt.Sprintf("%s:%s", cfg.mqttHost, cfg.mqttPort)
	mqtt := broker.New(address, handler, sRepo, cRepo, logger)

	errs <- mqtt.ListenAndServe()
}
