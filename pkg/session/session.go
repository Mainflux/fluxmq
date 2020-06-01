package session

import (
	"net"

	"go.uber.org/zap"
)

// Session represents MQTT Proxy session between client and broker.
type Session struct {
	clientID string
	conn     net.Conn
	logger   *zap.Logger
}

// New creates a new Session.
func New(clientID string, conn net.Conn, logger *zap.Logger) *Session {
	return &Session{
		clientID: clientID,
		conn:     conn,
		logger:   logger,
	}
}
