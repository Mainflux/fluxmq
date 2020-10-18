package server

import (
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/mainflux/fluxmq/auth"
	"github.com/mainflux/fluxmq/client"
	"github.com/mainflux/fluxmq/session"
	"go.uber.org/zap"
)

// Server is main MQTT proxy struct
type Server struct {
	address     string
	handler     auth.Handler
	logger      *zap.Logger
	dialer      net.Dialer
	sessionRepo *session.Repository
	clientRepo  *client.Repository
}

// New returns a new mqtt Server instance.
func New(address string, handler auth.Handler, sessionRepo *session.Repository, clientRepo *client.Repository, logger *zap.Logger) *Server {
	return &Server{
		address:     address,
		handler:     handler,
		logger:      logger,
		sessionRepo: sessionRepo,
		clientRepo:  clientRepo,
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

	s.logger.Info("Server exiting...")
	return nil
}

func (s Server) accept(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			s.logger.Warn("accept error " + err.Error())
			continue
		}

		s.logger.Info("accepted new client")
		go s.handle(conn)
	}
}

func (s Server) handle(conn net.Conn) {
	defer s.close(conn)

	// Check if MQTT CONNECT
	packet, err := packets.ReadPacket(conn)
	if err != nil {
		s.logger.Error("Read connect packet error: ", zap.Error(err))
		return
	}
	if packet == nil {
		s.logger.Error("Received nil packet")
		return
	}
	msg, ok := packet.(*packets.ConnectPacket)
	if !ok {
		s.logger.Error("Received msg that was not CONNECT")
		return
	}

	s.logger.Info("Received CONNECT", zap.String("client_id", msg.ClientIdentifier))

	connack := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	connack.SessionPresent = msg.CleanSession
	connack.ReturnCode = msg.Validate()

	// MQTT Client info
	ci := client.NewInfo(msg.ClientIdentifier, msg.Username, msg.Password)

	if connack.ReturnCode != packets.Accepted {
		if err := connack.Write(conn); err != nil {
			s.logger.Error("Send CONNACK error, ", zap.Error(err), zap.String("client_id", string(ci.ID)))
		}
		return
	}

	// Handle Auth
	if err := s.handler.AuthConnect(&ci); err != nil {
		connack.ReturnCode = packets.ErrRefusedNotAuthorised
		s.logger.Error("Auth error, ", zap.Error(err), zap.String("client_id", ci.ID))
		if err := connack.Write(conn); err != nil {
			s.logger.Error("Error witing response")
		}
		return
	}

	if err := connack.Write(conn); err != nil {
		s.logger.Error("Send CONNACK error, ", zap.Error(err), zap.String("client_id", ci.ID))
		return
	}

	// Last Will & Testament message
	lwt := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
	if msg.WillFlag {
		lwt.Qos = msg.WillQos
		lwt.TopicName = msg.WillTopic
		lwt.Retain = msg.WillRetain
		lwt.Payload = msg.WillMessage
		lwt.Dup = msg.Dup
	} else {
		lwt = nil
	}

	// ses, ok := s.sessionRepo.Sessions[ci.ID]
	// if !ok {
	// 	ses := session.New(ci.ID, conn, s.logger)
	// 	connack.SessionPresent = false
	// } else {
	// 	connack.SessionPresent = true
	// }

	c, ok := s.clientRepo.Clients[ci.ID]
	if !ok {
		c = client.New(ci, conn, s.logger)
	} else {
		// TODO: Client with this ID already exists - close old one
	}

	// Find Session for this client
	ses, ok := s.sessionRepo.Session(ci.ID)
	if !ok {
		ses = session.New(c, conn, msg.Keepalive, lwt, s.sessionRepo, s.logger)
		connack.SessionPresent = false
	} else {
		connack.SessionPresent = true
	}

	ses.ReadLoop()

}

func (s Server) close(conn net.Conn) {
	if err := conn.Close(); err != nil {
		s.logger.Warn("Error closing connection " + err.Error())
	}
}
