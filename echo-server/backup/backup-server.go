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

const BUFFER_BACKUP_FILE_SIZE = 10
const BUFFER_BACKUP = 1024

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
		emptySize := utils.FillString(strconv.FormatInt(0, 10), 10)
		client.Write([]byte(emptySize))
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
	fileSize := utils.FillString(strconv.FormatInt(fileInfo.Size(), 10), BUFFER_BACKUP_FILE_SIZE)
	
	client.Write([]byte(fileSize))
	log.Infof("Sending backup file size to connection ('%s', %s).", ip, port)
	
	sendBuffer := make([]byte, BUFFER_BACKUP)
	log.Infof("Start sending backup file to connection ('%s', %s).", ip, port)

	var currentByte int64 = 0
	for {
		idx := int(math.Ceil(float64(currentByte) / float64(BUFFER_BACKUP))) + 1
		log.Debugf("Start sending chunk #%d.", idx)

		sentBytes, err := backupFile.ReadAt(sendBuffer, currentByte)

		if sentBytes != 0 {
			_, err = client.Write(sendBuffer[:sentBytes])
			if err != nil {
				log.Errorf("Error sending chunk #%d, with %d bytes.", idx, sentBytes)
			}
			log.Debugf("Finish sending chunk #%d, with %d bytes.", idx, sentBytes)
		}

		if err != nil {
			if err == io.EOF {
				log.Debugf("Sending EOF in chunk #%d.", idx)
				break
			} else {
				log.Errorf("Error sending backup file to connection ('%s', %s).", ip, port)
			}
		}

		currentByte += BUFFER_BACKUP
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
