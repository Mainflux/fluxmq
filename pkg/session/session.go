package session

import (
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/mainflux/mainflux/errors"
	"go.uber.org/zap"
)

var (
	errBroker = errors.New("error between FluxMQ and MQTT broker")
	errClient = errors.New("error between FluxMQ and MQTT client")
)

type direction int

// Session represents MQTT Proxy session between client and broker.
type Session struct {
	logger  *zap.Logger
	conn    net.Conn
	handler Handler
	Client  Client
}

// New creates a new Session.
func New(conn net.Conn, handler Handler, logger *zap.Logger) *Session {
	return &Session{
		logger:  logger,
		conn:    conn,
		handler: handler,
	}
}

// Stream starts proxying traffic between client and broker.
func (s *Session) Stream() error {
	// In parallel read from client, send to broker
	// and read from broker, send to client.
	errs := make(chan error, 1)

	go s.stream(s.conn, errs)

	// Handle whichever error happens first.
	// The other routine won't be blocked when writing
	// to the errors channel because it is buffered.
	err := <-errs

	s.handler.Disconnect(&s.Client)
	return err
}

func (s *Session) stream(conn net.Conn, errs chan error) {

}

func (s *Session) notify(pkt packets.ControlPacket) {
	switch p := pkt.(type) {
	case *packets.ConnectPacket:
		s.handler.Connect(&s.Client)
	case *packets.PublishPacket:
		s.handler.Publish(&s.Client, &p.TopicName, &p.Payload)
	case *packets.SubscribePacket:
		s.handler.Subscribe(&s.Client, &p.Topics)
	case *packets.UnsubscribePacket:
		s.handler.Unsubscribe(&s.Client, &p.Topics)
	default:
		return
	}
}
