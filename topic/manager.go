// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package topic

import (
	"github.com/eclipse/paho.mqtt.golang/packets"
)

// Manager
type Manager interface {
	Subscribe(sub Subscription) (byte, error)
	Unsubscribe(sub Subscription) error
	Subscribers(topic []byte, qos byte, subs *[]interface{}, qoss *[]byte) error
	Retain(msg *packets.PublishPacket) error
	Retained(topic []byte, msgs *[]*packets.PublishPacket) error
	Close() error
}
