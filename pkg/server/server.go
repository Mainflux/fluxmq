package server

import (
	"io"
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/mainflux/fluxmq/pkg/session"
	"github.com/mainflux/mainflux/errors"
	"go.uber.org/zap"
)

// Server is main MQTT proxy struct
type Server struct {
	address string
	target  string
	handler session.Handler
	logger  *zap.Logger
	dialer  net.Dialer
}

// New returns a new mqtt Server instance.
func New(address, target string, handler session.Handler, logger *zap.Logger) *Server {
	return &Server{
		address: address,
		target:  target,
		handler: handler,
		logger:  logger,
	}
}

// MQTT server, this will block.
func (s Server) ListenAndServe() error {
	l, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	defer l.Close()

	// Acceptor loop
	s.accept(l)

	s.logger.Info("Server Exiting...")
	return nil
}

func (s Server) accept(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			s.logger.Warn("Accept error " + err.Error())
			continue
		}

		s.logger.Info("Accepted new client")
		go s.handle(conn)
	}
}

func (s Server) handle(conn net.Conn) {
	defer s.close(conn)

	// Check if MQTT CONNECT
	packet, err := packets.ReadPacket(conn)
	if err != nil {
		s.logger.Error("Error reading CONNECT")
		return
	}
	if packet == nil {
		s.logger.Error("Received nil packet")
		return
	}
	msg, ok := packet.(*packets.ConnectPacket)
	if !ok {
		s.logger.Error("Received msg that was not Connect")
		return
	}

	s.logger.Info("Read connect", zap.String("clientID", msg.ClientIdentifier))

	connack := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	connack.SessionPresent = msg.CleanSession
	connack.ReturnCode = msg.Validate()

	if connack.ReturnCode != packets.Accepted {
		err = connack.Write(conn)
		if err != nil {
			s.logger.Error("Send connack error", zap.Error(err), zap.String("clientID", msg.ClientIdentifier))
			return
		}
		return
	}

	ssn := session.New(conn, s.handler, s.logger)

	if err = ssn.Stream(); !errors.Contains(err, io.EOF) {
		//s.logger.Warn("Broken connection for client: " + s.Client.ID + " with error: " + err.Error())
	}
}

func (s Server) close(conn net.Conn) {
	if err := conn.Close(); err != nil {
		s.logger.Warn("Error closing connection " + err.Error())
	}
}
