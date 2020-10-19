// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package topic

type Subscription struct {
	ClientID string
	Topic    string
	QoS      byte
	Shared   bool
	Group    string
}

func NewSubscription(clientID string, topic string, qos byte, shared bool, group string) Subscription {
	return Subscription{
		ClientID: clientID,
		Topic:    topic,
		QoS:      qos,
		Shared:   shared,
		Group:    group,
	}
}
