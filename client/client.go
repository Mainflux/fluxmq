package client

import (
	"net"
	"regexp"

	"github.com/mainflux/fluxmq/topic"
	"go.uber.org/zap"
)

const (
	qos0 byte = iota
	qos1
	qos2
	qosFail = 0x80
)

var groupRegexp = regexp.MustCompile(`^\$share/([0-9a-zA-Z_-]+)/(.*)$`)

// Info stores basic MQTT client data.
// Separated in the struct so that it can be
// easily used as a param of Auth handler
type Info struct {
	ID       string
	Username string
	Password []byte
}

// Client stores complete MQTT client data
type Client struct {
	Info   Info
	Conn   net.Conn
	Subs   map[string]topic.Subscription
	Logger *zap.Logger
}

func New(info Info, conn net.Conn, logger *zap.Logger) Client {
	return Client{
		Info:   info,
		Conn:   conn,
		Subs:   make(map[string]topic.Subscription),
		Logger: logger,
	}
}

func NewInfo(clientID, username string, password []byte) Info {
	return Info{
		ID:       clientID,
		Username: username,
		Password: password,
	}
}
