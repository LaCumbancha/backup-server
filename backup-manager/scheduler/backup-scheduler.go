package scheduler

import (
	"io"
	"os"
	"net"
	"fmt"
	"math"
	"time"
	"strings"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/LaCumbancha/backup-server/backup-manager/utils"
	"github.com/LaCumbancha/backup-server/backup-manager/common"
)

const BUFFER_BACKUP_SIZE = 10
const BUFFER_BACKUP = 1024
const BUFFER_ETAG = 16
const BACKUP_TIME_WINDOW = 10

type BackupSchedulerConfig struct {
	Port 			string
	Storage 		*common.BackupStorage
}

type BackupRequest struct {
	Id 				string
	Ip 				string
	Port 			string
}

type BackupScheduler struct {
	port 			string
	storage 		*common.BackupStorage
	requests   		chan BackupRequest
}

func NewBackupScheduler(config BackupSchedulerConfig) *BackupScheduler {
	backupScheduler := &BackupScheduler {
		port:		config.Port,
		storage:	config.Storage,
	}

	return backupScheduler
}

func (bkpScheduler *BackupScheduler) updateBackupInformation(backupInfo common.BackupRegister, updatedTime time.Time) common.BackupRegister {
	freqDuration, _ := time.ParseDuration(backupInfo.Freq)				// Error ignored because it was already checked at registration.
	backupInfo.Next = updatedTime.Add(freqDuration)
	return backupInfo
}

func (bkpScheduler *BackupScheduler) checkBackups() chan BackupRequest {
	channel := make(chan BackupRequest)

	go func() {
		for {
			backups := bkpScheduler.storage.GetBackupClients()

			initialTime := time.Now()
			log.Debugf("Starting backup window at %s.", initialTime.String())

			var updatedBackups map[string]common.BackupRegister = make(map[string]common.BackupRegister)

			for backupId, backupInfo := range backups {
				updateTime := time.Now()

				if updateTime.After(backupInfo.Next) {
					log.Infof("Starting new backup for client %s at %s.", backupId, updateTime.String())

					// Sending backupID to request channel
					channel <- BackupRequest{
						Id: 			backupId,
						Ip:				backupInfo.Ip,
						Port:			backupInfo.Port,
					}

					// Update next backup information
					newBackupInfo := bkpScheduler.updateBackupInformation(backupInfo, updateTime)
					updatedBackups[backupId] = newBackupInfo

					log.Infof("Next backup for client %s setted at %s.", backupId, newBackupInfo.Next.String())
				} else {
					log.Debugf("Not in time for new backup for client %s.", backupId)
				}
			}

			bkpScheduler.storage.UpdateBackupClients(updatedBackups)

			endTime := time.Now()
			log.Debugf("Finishing backup window at %s.", initialTime.String())

			// Sleeping to complete the 5 seconds period.
			sleepTime := endTime.Add(time.Second * BACKUP_TIME_WINDOW).Sub(initialTime)
			log.Debugf("Setting new backup window at %s.", endTime.Add(time.Second * BACKUP_TIME_WINDOW))
			time.Sleep(sleepTime)
		}
	}()
	

	return channel
}

func (bkpScheduler *BackupScheduler) handleBackupConnection(backupRequest BackupRequest) {
	etag := bkpScheduler.storage.GenerateEtag(backupRequest.Id)
	log.Infof("Requesting new backup to client %s with etag '%s'", backupRequest.Id, etag)

	
	conn, err := net.Dial("tcp", backupRequest.Ip + ":" + backupRequest.Port)
	if err != nil {
		log.Errorf("Couldn't stablish connection with client %s", backupRequest.Ip)
		return
	}
	defer conn.Close()

	// Sending etag
	etagMessage := utils.FillString(etag, BUFFER_ETAG)
	
	conn.Write([]byte(etagMessage))
	log.Infof("Sending etag '%s' to backup connection ('%s', %s).", etag, backupRequest.Ip, backupRequest.Port)
	log.Debugf("Sending message '%b' to backup connection ('%s', %s).", etagMessage, backupRequest.Ip, backupRequest.Port)

	// Receiving backup
	bufferFileSize := make([]byte, BUFFER_BACKUP_SIZE)
	_, err = conn.Read(bufferFileSize)
	if err != nil {
		log.Errorf("Error receiving backup size from client %s.", backupRequest.Id)
	}

	fileSize, err := strconv.ParseInt(utils.UnfillString(bufferFileSize), 10, 64)
	if err != nil {
		log.Errorf("Error parsing backup file size from client %s.", backupRequest.Id)
		return
	}
	log.Infof("Received backup file (%d) size from connection ('%s', %s).", fileSize, backupRequest.Ip, backupRequest.Port)

	if fileSize == 0 {
		log.Infof("Current backup etag matches with current client %s etag, no information is transfered.", backupRequest.Id)
	} else {
		log.Infof("Starting new backup transfer. File size: %d.", fileSize)

		newFile, err := os.Create("Backup-" + fmt.Sprintf(time.Now().Format("20060102150405")))
		if err != nil {
			log.Errorf("Error creating new backup received from client %s.", backupRequest.Id)
		}
		defer newFile.Close()

		var receivedBytes int64
		
		for {
			idx := int(math.Ceil(float64(receivedBytes) / float64(BUFFER_BACKUP))) + 1
			log.Debugf("Start receiving chunk #%d.", idx)

			if (fileSize - receivedBytes) < BUFFER_BACKUP {
				log.Debugf("Receiving EOF in chunk #%d.", idx)
				io.CopyN(newFile, conn, (fileSize - receivedBytes))

				_, err = conn.Read(make([]byte, (receivedBytes+BUFFER_BACKUP)-fileSize))
				if err != nil {
					log.Errorf("Error receiving chunk %d from client %s.", idx, backupRequest.Id)
				}
				break
			}

			io.CopyN(newFile, conn, BUFFER_BACKUP)
			log.Debugf("Finish receiving chunk #%d.", idx)
			receivedBytes += BUFFER_BACKUP
		}

		log.Infof("Backup file received from connection ('%s', %s).", backupRequest.Ip, backupRequest.Port)
	}
	
}

func (bkpScheduler *BackupScheduler) Run() {
	listener, err := net.Listen("tcp", ":" + bkpScheduler.port)
	if listener == nil || err != nil {
		log.Fatalf("Error creating TCP BackupScheduler socket at port %s.", bkpScheduler.port, err)
	}

	// Start checking for new possible backups.
	bkpScheduler.requests = bkpScheduler.checkBackups()

	// Start parallel backup request.
	for {
		go bkpScheduler.handleBackupConnection(<-bkpScheduler.requests)
	}
}
