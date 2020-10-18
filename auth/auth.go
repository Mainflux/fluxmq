package auth

import "github.com/mainflux/fluxmq/client"

// AuthHandler is an interface for FluxMQ hooks
type Handler interface {
	// Authorization on client `CONNECT`
	// Each of the params are passed by reference, so that it can be changed
	AuthConnect(client *client.Info) error

	// Authorization on client `PUBLISH`
	// Topic is passed by reference, so that it can be modified
	AuthPublish(client *client.Info, topic *string, payload *[]byte) error

	// Authorization on client `SUBSCRIBE`
	// Topics are passed by reference, so that they can be modified
	AuthSubscribe(client *client.Info, topic *string) error

	// After client successfully connected
	Connect(client *client.Info)

	// After client successfully published
	Publish(client *client.Info, topic *string, payload *[]byte)

	// After client successfully subscribed
	Subscribe(client *client.Info, topics *[]string)

	// After client unsubscribed
	Unsubscribe(client *client.Info, topics *[]string)

	// Disconnect on connection with client lost
	Disconnect(client *client.Info)
}
