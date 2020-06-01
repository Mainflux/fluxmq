package client

import (
	"github.com/eclipse/paho.mqtt.golang/packets"
	"go.uber.org/zap"
)

// ClientInfo stores basic MQTT client data.
// Separated in the struct so that it can be
// easily used as a param of Auth handler
type ClientInfo struct {
	ID       string
	Username string
	Password []byte
}

// Client stores complete MQTT client data
type Client struct {
	info      ClientInfo
	keepalive uint16
	lwt       *packets.PublishPacket
	logger    *zap.Logger
}

func New(info ClientInfo, keepalive uint16, lwt *packets.PublishPacket, logger *zap.Logger) Client {
	return Client{
		info:      info,
		keepalive: keepalive,
		lwt:       lwt,
		logger:    logger,
	}
}

func NewInfo(clientID, username string, password []byte) ClientInfo {
	return ClientInfo{
		ID:       clientID,
		Username: username,
		Password: password,
	}
}

func (c Client) ReadLoop() {

}
