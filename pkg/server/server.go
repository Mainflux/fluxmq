// Copyright (c) Drasko DRASKOVIC
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"errors"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/drasko/fluxmq/message"
	"go.uber.org/zap"
)

const (
	defKeepAlive      = 300
	defConnectTimeout = 2
	defAckTimeout     = 20
	defTimeoutRetries = 3
)

var (
	ErrInvalidConnectionType error = errors.New("Invalid connection type")
	ErrInvalidSubscriber     error = errors.New("Invalid subscriber")
)

// Server is our main struct.
type Server struct {
	// The number of seconds to keep the connection live if there's no data.
	// If not set then default to 5 mins.
	keepAlive int

	// The number of seconds to wait for the CONNECT message before disconnecting.
	// If not set then default to 2 seconds.
	connectTimeout int

	// The number of seconds to wait for any ACK messages before failing.
	// If not set then default to 20 seconds.
	ackTimeout int

	// The number of times to retry sending a packet if ACK is not received.
	// If no set then default to 3 retries.
	timeoutRetries int

	// SessionsProvider is the session store that keeps all the Session objects.
	// This is the store to check if CleanSession is set to 0 in the CONNECT message.
	// If not set then default to "mem".
	sessionRepo string

	// TopicsProvider is the topic store that keeps all the subscription topics.
	// If not set then default to "mem".
	topicRepo string

	running      bool
	listener     net.Listener
	clients      map[uint64]*Client
	totalClients uint64
	done         chan bool
	start        time.Time
	host         string
	port         int
	logger       *zap.Logger
}

// New will setup a new server struct after parsing the options.
func New(host string, port int, logger *zap.Logger) *Server {
	s := &Server{
		host:           host,
		port:           port,
		done:           make(chan bool, 1),
		start:          time.Now(),
		logger:         logger,
		keepAlive:      defKeepAlive,
		connectTimeout: defConnectTimeout,
		timeoutRetries: defTimeoutRetries,
		ackTimeout:     defAckTimeout,
		timeoutRetries: defTimeoutRetries,
	}

	// For tracking clients
	s.sessions = make(map[uint64]*Session)

	return s
}

// Protected check on running state
func (s *Server) isRunning() bool {
	return s.running
}

// ListenAndServe of the server, this will block.
func (s *Server) ListenAndServe() error {
	// We are started
	s.running = true

	hp := net.JoinHostPort(s.host, strconv.Itoa(s.port))

	l, err := net.Listen("tcp", hp)
	if err != nil {
		s.logger.Error("Failed to get listener", zap.Error(err))
		return err
	}
	defer l.Close()

	// Setup state that can enable shutdown
	s.listener = l

	// Acceptor loop
	for {
		conn, err := l.Accept()
		if err != nil {
			s.logger.Error("Accept error", zap.Error(err))
			continue
		}

		s.logger.Info("Accepted new client")

		go handleConnection(conn)
	}

	s.logger.Info("Server Exiting..")
}

// HandleConnection is for the broker to handle an incoming connection from a client
func (s *Server) handleConnection(c io.Closer) (session *Session, err error) {
	if c == nil {
		return nil, ErrInvalidConnectionType
	}

	defer func() {
		if err != nil {
			c.Close()
		}
	}()

	conn, ok := c.(net.Conn)
	if !ok {
		return nil, ErrInvalidConnectionType
	}

	// To establish a connection, we must
	// 1. Read and decode the message.ConnectMessage from the wire
	// 2. If no decoding errors, then authenticate using username and password.
	//    Otherwise, write out to the wire message.ConnackMessage with
	//    appropriate error.
	// 3. If authentication is successful, then either create a new session or
	//    retrieve existing session
	// 4. Write out to the wire a successful message.ConnackMessage message

	// Read the CONNECT message from the wire, if error, then check to see if it's
	// a CONNACK error. If it's CONNACK error, send the proper CONNACK error back
	// to client. Exit regardless of error type.

	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(s.ConnectTimeout)))

	resp := message.NewConnackMessage()

	req, err := getConnectMessage(conn)
	if err != nil {
		if cerr, ok := err.(message.ConnackCode); ok {
			resp.SetReturnCode(cerr)
			resp.SetSessionPresent(false)
			writeMessage(conn, resp)
		}
		return nil, err
	}

	if req.KeepAlive() == 0 {
		req.SetKeepAlive(minKeepAlive)
	}

	session := session.New(s.connectTimeout, s.ackTimeout, s.timeoutRetries)

	resp.SetReturnCode(message.ConnectionAccepted)

	if err = writeMessage(c, resp); err != nil {
		return nil, err
	}

	svc.inStat.increment(int64(req.Len()))
	svc.outStat.increment(int64(resp.Len()))

	if err := session.Establish(); err != nil {
		session.Close()
		return nil, err
	}

	s.logger.Info("(%s) server/handleConnection: Connection established.", session.id())

	return session, nil
}
