package backup

import (
	"io"
	"net"
	"bufio"

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
			log.Infof("Connection ('%s', %s) closed.", ip, port)
			break
		} else if err != nil {
			log.Errorf("Couldn't read line", err)
		}

		newEtag := string(line)
		log.Infof("Backup request received from connection ('%s', %s). E-Tag: %s", ip, port, newEtag)
		backupServer.handleBackup(client, newEtag)
	}
}

func (backupServer *BackupServer) handleBackup(client net.Conn, newEtag string) {
	backupServer.storage.GenerateBackup()
	//oldEtag, compressedFiles := backupServer.storage.GenerateBackup()
	//defer client.Close()

	//if oldEtag == newEtag {
	//	log.Infof("There's no difference beetween current version and last sent. Backup skipped.")
	//	client.Write([]byte("UNMODIFIED"))
	//} else {
	//	log.Infof("Sending new backup.")
	//}
}

func (backupServer *BackupServer) Run() {
	// Create server
	listener, err := net.Listen("tcp", ":" + backupServer.port)
	if listener == nil || err != nil {
		log.Fatalf("[SERVER] Error creating TCP server socket at port %s.", backupServer.port)
	}

	backupServer.listenBackups(listener)
}
