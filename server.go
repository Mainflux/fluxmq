package fluxmq

import (
	"net"
	"strconv"
	"time"

	"go.uber.org/zap"
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
	s.clients = make(map[uint64]*Client)

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

		client := &Client{
			conn:   conn,
			server: s,
		}
		go client.listen()
	}

	s.logger.Info("Server Exiting..")
}
