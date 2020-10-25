package scheduler

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/LaCumbancha/backup-server/backup-manager/common"
)

const BACKUP_WINDOW = 10

type BackupSchedulerConfig struct {
	Storage 		*common.BackupStorage
}

type BackupScheduler struct {
	storage 		*common.BackupStorage
}

func NewBackupScheduler(config BackupSchedulerConfig) *BackupScheduler {
	backupScheduler := &BackupScheduler {
		storage:	config.Storage,
	}

	return backupScheduler
}

func (bkpScheduler *BackupScheduler) updateBackupInformation(backupInfo common.BackupRegister, updatedTime time.Time) common.BackupRegister {
	freqDuration, _ := time.ParseDuration(backupInfo.Freq)				// Error ignored because it was already checked at registration.
	backupInfo.Next = updatedTime.Add(freqDuration)
	return backupInfo
}

func (bkpScheduler *BackupScheduler) Run() {
	for {
		backups := bkpScheduler.storage.GetBackupClients()

		initialTime := time.Now()
		log.Debugf("Starting backup window at %s.", initialTime.String())

		var updatedBackups map[string]common.BackupRegister = make(map[string]common.BackupRegister)

		for backupId, backupInfo := range backups {
			updateTime := time.Now()

			if updateTime.After(backupInfo.Next) {
				log.Infof("Starting new backup for client %s at %s.", backupId, updateTime.String())

				// Execute backup
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
		sleepTime := endTime.Add(time.Second * BACKUP_WINDOW).Sub(initialTime)
		log.Debugf("Setting new backup window at %s.", endTime.Add(time.Second * BACKUP_WINDOW))
		time.Sleep(sleepTime)
	}
}
