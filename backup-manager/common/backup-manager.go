package common

import (
	"io"
	"fmt"
	"net"
	"bufio"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/LaCumbancha/backup-server/backup-manager/utils"
)

type BackupManagerConfig struct {
	Port 			string
	StoragePath		string
}

type BackupManager struct {
	port 			string
	storage 		*BackupStorage
	conns   		chan net.Conn
}

func NewBackupManager(config BackupManagerConfig) *BackupManager {
	backupStorage := &BackupStorage {
		path: 		config.StoragePath,
	}

	go backupStorage.BuildBackupStructure()

	backupManager := &BackupManager {
		port: 		config.Port,
		storage:	backupStorage,
	}

	return backupManager
}

// Accepting connections
func (bkpManager *BackupManager) acceptConnections(listener net.Listener) chan net.Conn {
	channel := make(chan net.Conn)

	go func() {
		for {
			client, err := listener.Accept()

			if client == nil || err != nil {
				//log.Errorf("Couldn't accept client", err)
				fmt.Printf("Couldn't accept client")
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

// Saving new backup client
func (bkpManager *BackupManager) handleConnections(client net.Conn) {
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

		var backupClient BackupClient
		json.Unmarshal([]byte(strLine), &backupClient)

		if bkpManager.validateBackupRequest(backupClient) {
			outputMessage := bkpManager.processBackupRequest(backupClient)
			utils.SocketWrite(outputMessage, client)
		} else {
			outputMessage := fmt.Sprintf("Some request mandatory fields are missing. Message received: %s", strLine)
			utils.SocketWrite(outputMessage, client)
		}
	}
}

func (bkpManager *BackupManager) validateBackupRequest(backupRequest BackupClient) bool {
	if backupRequest.Ip == "" || backupRequest.Port == "" || backupRequest.Path == "" || backupRequest.Freq == "" {
		log.Errorf("Error receiving some REGISTER mandatory fields. IP: %s; Port: %s; Path: '%s'; Frequency: %s.", backupRequest.Ip, backupRequest.Port, backupRequest.Path, backupRequest.Freq)
		return false
	}

	return true
}

func (bkpManager *BackupManager) processBackupRequest(backupRequest BackupClient) string {
	bkpManager.storage.UpdateBackupInfo(backupRequest)
	log.Infof("New backup client request, with IP %s, port %s, path %s and frequency %s", backupRequest.Ip, backupRequest.Port, backupRequest.Path, backupRequest.Freq)
	return "New backup client request successfully added.\n"
}

func (bkpManager *BackupManager) Run() {
	listener, err := net.Listen("tcp", ":" + bkpManager.port)
	if listener == nil || err != nil {
		log.Fatalf("Error creating TCP BackupManager socket at port %s.", bkpManager.port, err)
	}

	// Start processing connections
	bkpManager.conns = bkpManager.acceptConnections(listener)

	// Start parallel messages echo
	for {
		go bkpManager.handleConnections(<-bkpManager.conns)
	}
}
