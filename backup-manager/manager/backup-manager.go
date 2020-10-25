package manager

import (
	"io"
	"fmt"
	"net"
	"bufio"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/LaCumbancha/backup-server/backup-manager/utils"
	"github.com/LaCumbancha/backup-server/backup-manager/common"
)

const ADD_BACKUP = "REGISTER"
const QUERY_BACKUP = "QUERY"
const REMOVE_BACKUP = "UNREGISTER"

type BackupManagerConfig struct {
	Port 			string
	Storage 		*common.BackupStorage
}

type BackupManager struct {
	port 			string
	storage 		*common.BackupStorage
	conns   		chan net.Conn
}

func NewBackupManager(config BackupManagerConfig) *BackupManager {
	backupManager := &BackupManager {
		port: 		config.Port,
		storage:	config.Storage,
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

		var backupRequest common.BackupRequest
		json.Unmarshal([]byte(strLine), &backupRequest)

		if bkpManager.validateBackupRequest(backupRequest) {
			outputMessage := bkpManager.processBackupRequest(backupRequest)
			utils.SocketWrite(outputMessage, client)
		} else {
			outputMessage := fmt.Sprintf("Some mandatory fields are missing. Message received: %s", strLine)
			utils.SocketWrite(outputMessage, client)
		}
	}
}

func (bkpManager *BackupManager) validateBackupRequest(backupRequest common.BackupRequest) bool {
	switch backupRequest.Verb {
	case ADD_BACKUP:
		backupRegister := backupRequest.Args

		if backupRegister.Ip == "" || backupRegister.Port == "" || backupRegister.Path == "" || backupRegister.Freq == "" {
			log.Errorf("Error receiving some REGISTER mandatory fields. IP: '%s'; Port: '%s'; Path: '%s'; Frequency: '%s'.", backupRegister.Ip, backupRegister.Port, backupRegister.Path, backupRegister.Freq)
			return false
		}
	case QUERY_BACKUP:
		backupQuery := backupRequest.Args

		if backupQuery.Ip == "" || backupQuery.Port == "" || backupQuery.Path == "" {
			log.Errorf("Error receiving some QUERY mandatory fields. IP: '%s'; Port: '%s'; Path: '%s'", backupQuery.Ip, backupQuery.Port, backupQuery.Path)
			return false
		}
	case REMOVE_BACKUP:
		backupUnregister := backupRequest.Args

		if backupUnregister.Ip == "" || backupUnregister.Port == "" || backupUnregister.Path == "" {
			log.Errorf("Error receiving some UNREGISTER mandatory fields. IP: '%s'; Port: '%s'; Path: '%s'", backupUnregister.Ip, backupUnregister.Port, backupUnregister.Path)
			return false
		}
	default:
		log.Errorf("Verb not recognized: %s.", backupRequest.Verb)
		return false
	}
	
	return true
}

func (bkpManager *BackupManager) processBackupRequest(backupRequest common.BackupRequest) string {
	switch backupRequest.Verb {
	case ADD_BACKUP:
		backupRegister := backupRequest.Args

		log.Infof("New REGISTER backup client request received, with IP '%s', port '%s', path '%s' and frequency '%s'.", backupRegister.Ip, backupRegister.Port, backupRegister.Path, backupRegister.Freq)
		return bkpManager.storage.AddBackupClient(backupRegister)
	case QUERY_BACKUP:
		backupQuery := backupRequest.Args

		// TODO
		log.Infof("New QUERY request received, for backup with IP '%s', port '%s' and path '%s'.", backupQuery.Ip, backupQuery.Port, backupQuery.Path)
		return "Query result!\n"
	case REMOVE_BACKUP:
		backupRegister := backupRequest.Args

		log.Infof("New UNREGISTER backup client request received, with IP '%s', port '%s' and path '%s'.", backupRegister.Ip, backupRegister.Port, backupRegister.Path)
		return bkpManager.storage.RemoveBackupClient(backupRegister)
	default:
		log.Fatalf("Flow forbidden.")
	}

	return "Flow forbidden."
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
