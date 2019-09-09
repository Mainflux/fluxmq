// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package server

import "github.com/mainflux/fluxmq/pkg/session"

// Client is a library implementation of the MQTT client that, as best it can, complies
// with the MQTT 3.1 and 3.1.1 specs.
type Client struct {
	// The number of seconds to keep the connection live if there's no data.
	// If not set then default to 5 mins.
	KeepAlive int

	// The number of seconds to wait for the CONNACK message before disconnecting.
	// If not set then default to 2 seconds.
	ConnectTimeout int

	// The number of seconds to wait for any ACK messages before failing.
	// If not set then default to 20 seconds.
	AckTimeout int

	// The number of times to retry sending a packet if ACK is not received.
	// If no set then default to 3 retries.
	TimeoutRetries int

	session *session.Session
}
