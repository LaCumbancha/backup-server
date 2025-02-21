package backup

import (
	"os"
	"io"
	"net"
	"math"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/LaCumbancha/backup-server/echo-server/common"
	"github.com/LaCumbancha/backup-server/echo-server/utils"
)

const BUFFER_ETAG = 64
const BUFFER_BACKUP_PATH = 256
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

		ip, port := utils.ParseAddress(client.RemoteAddr().String())
		log.Infof("Got backup connection from ('%s', %s).", ip, port)

		etagBuffer := make([]byte, BUFFER_ETAG)
		_, err = client.Read(etagBuffer)
		if err != nil {
			log.Errorf("Error receiving etag from backup scheduler at ('%s', %s).", ip, port, err)
			return
		}

		receivedEtag := utils.UnfillString(etagBuffer)
		log.Infof("Backup request received from connection ('%s', %s). E-Tag: %s", ip, port, receivedEtag)
		backupServer.handleBackup(client, receivedEtag)
	}
}

func (backupServer *BackupServer) handleBackup(client net.Conn, receivedEtag string) {
	ip, port := utils.ParseAddress(client.RemoteAddr().String())

	backupPath := make([]byte, BUFFER_BACKUP_PATH)
	_, err := client.Read(backupPath)
	if err != nil {
		log.Errorf("Error receiving path from backup scheduler at ('%s', %s).", ip, port, err)
		return
	}
	receivedPath := utils.UnfillString(backupPath)
	log.Infof("Path requested to backup from connection (%s, %s): %s.", ip, port, receivedPath)

	currentEtag, backupFile := backupServer.storage.GenerateBackup(receivedPath)
	defer client.Close()

	if backupFile == nil {
		errorSize := utils.FillString(strconv.FormatInt(-1, 10), 10)
		client.Write([]byte(errorSize))
	}
	if currentEtag == receivedEtag {
		backupFile.Close()
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
	fileSize := strconv.FormatInt(fileInfo.Size(), 10)
	fileSizeMessage := utils.FillString(fileSize, BUFFER_BACKUP_FILE_SIZE)
	
	client.Write([]byte(fileSizeMessage))
	log.Infof("Sending backup file size to connection ('%s', %s).", ip, port)
	
	sendBuffer := make([]byte, BUFFER_BACKUP)
	log.Infof("Start sending backup file (size %s) to connection ('%s', %s).", fileSize, ip, port)

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
				log.Errorf("Error sending backup file to connection ('%s', %s).", ip, port, err)
			}
		}

		currentByte += BUFFER_BACKUP
	}

	backupFile.Close()
	log.Infof("Backup file sent to connection ('%s', %s).", ip, port)
}

func (backupServer *BackupServer) Run() {
	// Create server
	listener, err := net.Listen("tcp", ":" + backupServer.port)
	if listener == nil || err != nil {
		log.Fatalf("[SERVER] Error creating TCP server socket at port %s.", backupServer.port)
	}

	backupServer.listenBackups(listener)
}
