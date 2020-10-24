package server

import (
	"io"
	"net"
	"bufio"

	log "github.com/sirupsen/logrus"

	"github.com/LaCumbancha/backup-server/echo-server/common"
	"github.com/LaCumbancha/backup-server/echo-server/utils"
)

type EchoServer struct {
	port 		string
	storage 	*common.StorageManager
	conns   	chan net.Conn
}

func NewEchoServer(config common.ServerConfig) *EchoServer {
	storageManager := &common.StorageManager {
		Path: 		config.StoragePath,
	}

	go storageManager.BuildStorage()

	server := &EchoServer {
		port: 		config.Port,
		storage:	storageManager,
	}
	
	return server
}

func (echoServer *EchoServer) acceptConnections(listener net.Listener) chan net.Conn {
	channel := make(chan net.Conn)

	go func() {
		for {
			client, err := listener.Accept()

			if client == nil || err != nil {
				log.Errorf("Couldn't accept client", err)
				continue
			}

			ip, port := utils.ParseAddress(client.RemoteAddr().String())
			log.Infof("Got connection from ('%s', %s).", ip, port)

			channel <- client
			log.Infof("Proceed to accept new connections.")
		}
	}()

	return channel
}

func (echoServer *EchoServer) handleConnections(client net.Conn) {
	buffer := bufio.NewReader(client)
	ip, port := utils.ParseAddress(client.RemoteAddr().String())

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

		echoServer.storage.UpdateStorage(strLine)
		client.Write(line)
	}
}

func (echoServer *EchoServer) Run() {
	// Create server
	listener, err := net.Listen("tcp", ":" + echoServer.port)
	if listener == nil || err != nil {
		log.Fatalf("[SERVER] Error creating TCP server socket at port %s.", echoServer.port)
	}

	// Start processing connections
	echoServer.conns = echoServer.acceptConnections(listener)

	// Start parallel messages echo
	for {
		go echoServer.handleConnections(<-echoServer.conns)
	}
}
