package fluxmq

import (
	"net"
	"strconv"
	"time"

	"github.com/drasko/fluxmq/message"
	"go.uber.org/zap"
)

var (
	ErrInvalidConnectionType  error = errors.New("Invalid connection type")
	ErrInvalidSubscriber      error = errors.New("Invalid subscriber")
)

// Server is our main struct.
type Server struct {
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
		host:   host,
		port:   port,
		done:   make(chan bool, 1),
		start:  time.Now(),
		logger: logger,
	}

	// For tracking clients
	s.sessions = make(map[uint64]*Session)

	return s
}

// Protected check on running state
func (s *Server) isRunning() bool {
	return s.running
}

// Start up the server, this will block.
func (s *Server) Start() {
	s.running = true

	// Wait for clients.
	s.acceptLoop()
}

func (s *Server) acceptLoop() {
	hp := net.JoinHostPort(s.host, strconv.Itoa(s.port))

	l, err := net.Listen("tcp", hp)
	if err != nil {
		s.logger.Error("Failed to start", zap.Error(err))
		return
	}
	defer l.Close()

	// Setup state that can enable shutdown
	s.listener = l

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

	s.configure()

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

	session := &Session{
		id:     atomic.AddUint64(&gsvcid, 1),
		client: false,

		keepAlive:      int(req.KeepAlive()),
		connectTimeout: s.ConnectTimeout,
		ackTimeout:     s.AckTimeout,
		timeoutRetries: s.TimeoutRetries,

		conn:      conn,
	}

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

func (s *Server) configure() {
	var err error

	if s.KeepAlive == 0 {
		s.KeepAlive = DefaultKeepAlive
	}

	if s.ConnectTimeout == 0 {
		s.ConnectTimeout = DefaultConnectTimeout
	}

	if s.AckTimeout == 0 {
		s.AckTimeout = DefaultAckTimeout
	}

	if s.TimeoutRetries == 0 {
		s.TimeoutRetries = DefaultTimeoutRetries
	}

	return
}
