package common

import (
	"bufio"
	"net"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ServerConfig Configuration used by the server
type ServerConfig struct {
	Port string
}

// Server Entity that encapsulates how
type Server struct {
	config 	ServerConfig
	conns   chan net.Conn
}

// NewServer Initializes a new server receiving the configuration as a parameter
func NewServer(config ServerConfig) *Server {
	server := &Server {
		config: config,
	}
	return server
}

func ParseAddress(address string) (string, string) {
	split := strings.Split(address, ":")
	ip := split[0]
	port := split[1]

	return ip, port
}

// Accepting connections
func (s *Server) acceptConnections(listener net.Listener) chan net.Conn {
	channel := make(chan net.Conn)

	go func() {
		for {
			client, err := listener.Accept()

			if client == nil || err != nil {
				log.Errorf("Couldn't accept client", err)
				continue
			}

			ip, port := ParseAddress(client.RemoteAddr().String())
			log.Infof("Got connection from ('%s', %s).", ip, port)

			channel <- client
			log.Infof("Proceed to accept new connections.")
		}
	}()

	return channel
}

func (s *Server) handleConnections(client net.Conn) {
	buffer := bufio.NewReader(client)

	for {
		line, err := buffer.ReadBytes('\n')

		if err != nil {
			break
		}

		ip, port := ParseAddress(client.RemoteAddr().String())
		log.Infof("Message received from connection ('%s', %s). Msg: %s", ip, port, string(line))

		client.Write(line)
	}
}

// Run start listening for client messages
func (s *Server) Run() {
	// Create server
	listener, _ := net.Listen("tcp", ":" + s.config.Port)
	if listener == nil {
		log.Fatalf("[SERVER] Error creating TCP server socket at port %s.", s.config.Port)
	}

	// Start processing connections
	s.conns = s.acceptConnections(listener)

	// Start parallel messages echo
	for {
		go s.handleConnections(<-s.conns)
	}
}
