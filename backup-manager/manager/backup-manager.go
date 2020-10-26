package manager

import (
	"io"
	"os"
	"fmt"
	"net"
	"math"
	"bufio"
	"strconv"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/LaCumbancha/backup-server/backup-manager/utils"
	"github.com/LaCumbancha/backup-server/backup-manager/common"
)

const BUFFER_BACKUP_LOG = 1024
const BUFFER_BACKUP_LOG_SIZE = 10

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
	defer client.Close()
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
			bkpManager.processBackupRequest(client, backupRequest)
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

func (bkpManager *BackupManager) processBackupRequest(client net.Conn, backupRequest common.BackupRequest) {
	switch backupRequest.Verb {
	case ADD_BACKUP:
		backupRegister := backupRequest.Args
		log.Infof("New REGISTER backup client request received, with IP '%s', port '%s', path '%s' and frequency '%s'.", backupRegister.Ip, backupRegister.Port, backupRegister.Path, backupRegister.Freq)

		outputMessage := bkpManager.storage.AddBackupClient(backupRegister)
		utils.SocketWrite(outputMessage, client)
	case QUERY_BACKUP:
		backupQuery := backupRequest.Args
		log.Infof("New QUERY request received, for backup with IP '%s', port '%s' and path '%s'.", backupQuery.Ip, backupQuery.Port, backupQuery.Path)

		backupLog, backupLogSize := bkpManager.storage.RetrieveBackupLog(backupQuery)

		if backupLogSize < 0 {
			fileSizeMessage := utils.FillString("0", BUFFER_BACKUP_LOG_SIZE)
			client.Write([]byte(fileSizeMessage))

			ip, port := utils.ParseAddress(client.RemoteAddr().String())
			log.Infof("Sending empyt log file size to connection ('%s', %s).", ip, port)

			utils.SocketWrite("There was an error retrieving the requested backup log. Try again later.", client)
		} else {
			bkpManager.sendBackupLog(client, backupLog, backupLogSize)
		}
	case REMOVE_BACKUP:
		backupRegister := backupRequest.Args
		log.Infof("New UNREGISTER backup client request received, with IP '%s', port '%s' and path '%s'.", backupRegister.Ip, backupRegister.Port, backupRegister.Path)

		outputMessage := bkpManager.storage.RemoveBackupClient(backupRegister)
		utils.SocketWrite(outputMessage, client)
	default:
		log.Fatalf("Flow forbidden.")
	}
}

func (bkpManager *BackupManager) sendBackupLog(client net.Conn, file *os.File, size int64) {
	ip, port := utils.ParseAddress(client.RemoteAddr().String())
	fileSize := strconv.FormatInt(size, 10)
	fileSizeMessage := utils.FillString(fileSize, BUFFER_BACKUP_LOG_SIZE)
	
	client.Write([]byte(fileSizeMessage))
	log.Infof("Sending backup log file size (%d) to connection ('%s', %s).", fileSize, ip, port)
	
	sendBuffer := make([]byte, BUFFER_BACKUP_LOG)
	log.Infof("Start sending backup log file (size %d) to connection ('%s', %s).", fileSize, ip, port)

	var currentByte int64 = 0
	for {
		idx := int(math.Ceil(float64(currentByte) / float64(BUFFER_BACKUP_LOG))) + 1
		log.Debugf("Start sending chunk #%d.", idx)

		sentBytes, err := file.ReadAt(sendBuffer, currentByte)

		if sentBytes != 0 {
			_, err = client.Write(sendBuffer[:sentBytes])
			if err != nil {
				log.Errorf("Error sending chunk #%d, with %d bytes.", idx, sentBytes, err)
			}
			log.Debugf("Finish sending chunk #%d, with %d bytes.", idx, sentBytes)
		}

		if err != nil {
			if err == io.EOF {
				log.Debugf("Sending EOF in chunk #%d.", idx)
			} else {
				log.Errorf("Error sending backup log file to connection ('%s', %s).", ip, port, err)
			}
			break
		}

		currentByte += BUFFER_BACKUP_LOG
	}

	file.Close()
	log.Infof("Backup log file sent to connection ('%s', %s).", ip, port)
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
