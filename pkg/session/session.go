// Copyright (c) Drasko DRASKOVIC
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"bufio"
	"io"
	"net"
	"sync/atomic"
)

type Session struct {
	// The ID of this session, it's not related to the Client ID, just a number that's
	// incremented for every new session.
	id uint64

	// The number of seconds to keep the connection live if there's no data.
	// If not set then default to 5 mins.
	keepAlive int

	// The number of seconds to wait for the CONNACK message before disconnecting.
	// If not set then default to 2 seconds.
	connectTimeout int

	// The number of seconds to wait for any ACK messages before failing.
	// If not set then default to 20 seconds.
	ackTimeout int

	// The number of times to retry sending a packet if ACK is not received.
	// If no set then default to 3 retries.
	timeoutRetries int

	// Network connection for this session
	conn io.Closer

	// Whether this is service is closed or not.
	closed int64
}

func (s *Session) New(connectTout, ackTout, toutRetries int) error {
	return &Session{
		id:     atomic.AddUint64(&gsvcid, 1),
		client: false,

		keepAlive:      int(req.KeepAlive()),
		connectTimeout: connectTout,
		ackTimeout:     ackTout,
		timeoutRetries: toutRetries,

		conn: conn,
	}
}

func (s *Session) Establish() error {
	// TODO
	return nil
}

func (s *Session) Close() {
	// TODO
}

// Read client data from channel
func (s *Session) Receive() {
	reader := bufio.NewReader(s.conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			s.conn.Close()
			//c.server.onClientConnectionClosed(c, err)
			return
		}
		//c.server.onNewMessage(c, message)
		println(message)
	}
}

// Send text message to client
func (s *Session) Send(message string) error {
	_, err := s.conn.Write([]byte(message))
	return err
}

// Send bytes to client
func (s *Session) SendBytes(b []byte) error {
	_, err := s.conn.Write(b)
	return err
}

func (s *Session) Conn() net.Conn {
	return s.conn
}
