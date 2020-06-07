package auth

import (
	"github.com/mainflux/fluxmq/pkg/client"
)

// Handler is an interface for FluxMQ hooks
type Handler interface {
	// Authorization on client `CONNECT`
	// Each of the params are passed by reference, so that it can be changed
	AuthConnect(client *client.ClientInfo) error

	// Authorization on client `PUBLISH`
	// Topic is passed by reference, so that it can be modified
	AuthPublish(client *client.ClientInfo, topic *string, payload *[]byte) error

	// Authorization on client `SUBSCRIBE`
	// Topics are passed by reference, so that they can be modified
	AuthSubscribe(client *client.ClientInfo, topics *[]string) error

	// After client successfully connected
	Connect(client *client.ClientInfo)

	// After client successfully published
	Publish(client *client.ClientInfo, topic *string, payload *[]byte)

	// After client successfully subscribed
	Subscribe(client *client.ClientInfo, topics *[]string)

	// After client unsubscribed
	Unsubscribe(client *client.ClientInfo, topics *[]string)

	// Disconnect on connection with client lost
	Disconnect(client *client.ClientInfo)
}
