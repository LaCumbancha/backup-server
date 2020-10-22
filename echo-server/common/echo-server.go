package common

import (
	"io"
	"os"
	"net"
	"bufio"
	"strings"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

type EchoServerConfig struct {
	Port 		string
	Storage		string
}

type EchoServer struct {
	config 		EchoServerConfig
	conns   	chan net.Conn
}

func NewEchoServer(config EchoServerConfig) *EchoServer {
	server := &EchoServer {
		config: config,
	}

	server.buildStorageFile()
	return server
}

func (es *EchoServer) buildStorageFile() {
	err := os.MkdirAll(filepath.Dir(es.config.Storage), os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating EchoStorage directory.", err)
	}

	file, err := os.Create(es.config.Storage)
	if err != nil {
		log.Fatalf("Error creating EchoStorage file.", err)
	}

	file.Close()
}

func ParseAddress(address string) (string, string) {
	split := strings.Split(address, ":")
	ip := split[0]
	port := split[1]

	return ip, port
}

func (es *EchoServer) acceptConnections(listener net.Listener) chan net.Conn {
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

func (es *EchoServer) handleConnections(client net.Conn) {
	buffer := bufio.NewReader(client)
	ip, port := ParseAddress(client.RemoteAddr().String())

	for {
		line, err := buffer.ReadBytes('\n')

		if err == io.EOF {
			log.Infof("Connection ('%s', %s) closed.", ip, port)
			break
		} else if err != nil {
			log.Errorf("Couldn't read line", err)
		}

		strLine := string(line)
		log.Infof("Message received from connection ('%s', %s). Msg: %s", ip, port, strLine)

		es.updateStoredFile(strLine)
		client.Write(line)
	}
}

func (es *EchoServer) updateStoredFile(line string) {
	file, err := os.OpenFile(es.config.Storage, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
        log.Fatalf("Error opening storage file.", err)
    }

    defer file.Close()
 
    _, err = file.WriteString(line)
    if err != nil {
        log.Fatalf("Error writing storage file.", err)
    }

    log.Infof("New message stored in server: %s.", line)
}

func (es *EchoServer) Run() {
	// Create server
	listener, err := net.Listen("tcp", ":" + es.config.Port)
	if listener == nil || err != nil {
		log.Fatalf("[SERVER] Error creating TCP server socket at port %s.", es.config.Port)
	}

	// Start processing connections
	es.conns = es.acceptConnections(listener)

	// Start parallel messages echo
	for {
		go es.handleConnections(<-es.conns)
	}
}
