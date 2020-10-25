package backup

import (
	"os"
	"io"
	"net"
	"math"
	"bufio"
	"strings"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/LaCumbancha/backup-server/echo-server/common"
	"github.com/LaCumbancha/backup-server/echo-server/utils"
)

type BackupServer struct {
	port 		string
	storage 	*common.StorageManager
}

func NewBackupServer(config common.ServerConfig) *BackupServer {
	echoStorage := &common.StorageManager {
		Path: 		config.StoragePath,
	}

	server := &BackupServer {
		port: 		config.Port,
		storage:	echoStorage,
	}
	
	return server
}

func (backupServer *BackupServer) listenBackups(listener net.Listener) {
	for {
		client, err := listener.Accept()

		if client == nil || err != nil {
			log.Errorf("Couldn't accept backup client.", err)
			continue
		}

		buffer := bufio.NewReader(client)
		ip, port := utils.ParseAddress(client.RemoteAddr().String())
		log.Infof("Got backup connection from ('%s', %s).", ip, port)

		line, err := buffer.ReadBytes('\n')
		if err == io.EOF {
			log.Infof("Backup connection ('%s', %s) closed.", ip, port)
			break
		} else if err != nil {
			log.Errorf("Couldn't read line", err)
		}

		receivedEtag := strings.TrimSuffix(string(line), "\n")
		log.Infof("Backup request received from connection ('%s', %s). E-Tag: %s", ip, port, receivedEtag)
		backupServer.handleBackup(client, receivedEtag)
	}
}

func (backupServer *BackupServer) handleBackup(client net.Conn, receivedEtag string) {
	currentEtag, backupFile := backupServer.storage.GenerateBackup()
	defer client.Close()

	if currentEtag == receivedEtag {
		log.Infof("There's no difference beetween current version and last sent. Backup skipped.")
		client.Write([]byte("UNMODIFIED"))
	} else {
		log.Infof("Sending new backup with E-Tag: %s", currentEtag)
		backupServer.sendBackupFile(client, backupFile)
	}
}

func (backupServer *BackupServer) sendBackupFile(client net.Conn, backupFile *os.File) {
	fileInfo, err := backupFile.Stat()
	if err != nil {
		log.Errorf("Couldn't retrieve backup file information. Aborting backup.", err)
		return
	}

	ip, port := utils.ParseAddress(client.RemoteAddr().String())
	fileSize := utils.FillString(strconv.FormatInt(fileInfo.Size(), 10), 10)
	fileName := utils.FillString(fileInfo.Name(), 64)
	
	client.Write([]byte(fileSize))
	log.Infof("Sending backup file size to connection ('%s', %s).", ip, port)
	
	client.Write([]byte(fileName))
	log.Infof("Sending backup file name to connection ('%s', %s).", ip, port)
	
	sendBuffer := make([]byte, common.BUFFER_SIZE)
	log.Infof("Start sending backup file to connection ('%s', %s).", ip, port)

	var currentByte int64 = 0
	for {
		idx := int(math.Ceil(float64(currentByte) / float64(common.BUFFER_SIZE))) + 1
		log.Debugf("Start sending package #%d.", idx)

		sentBytes, err := backupFile.ReadAt(sendBuffer, currentByte)

		if sentBytes != 0 {
			client.Write(sendBuffer[:sentBytes])
			log.Debugf("Finish sending package #%d, with %d bytes.", idx, sentBytes)
		}

		if err != nil {
			if err == io.EOF {
				log.Debugf("Sending EOF in package #%d.", idx)
				break
			} else {
				log.Errorf("Error sending backup file to connection ('%s', %s).", ip, port)
			}
		}

		currentByte += common.BUFFER_SIZE
	}

	log.Infof("Backup file sent to connection ('%s', %s).", ip, port)

	defer backupFile.Close()
}

func (backupServer *BackupServer) Run() {
	// Create server
	listener, err := net.Listen("tcp", ":" + backupServer.port)
	if listener == nil || err != nil {
		log.Fatalf("[SERVER] Error creating TCP server socket at port %s.", backupServer.port)
	}

	backupServer.listenBackups(listener)
}
