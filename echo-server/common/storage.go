package common

import (
	"io"
	"os"
	"fmt"
	"crypto/sha256"

	log "github.com/sirupsen/logrus"
)

const INFO_FILE = "Data.info"
const BACKUP_FILE = "Backup.tar.gz"

type StorageConfig struct {
	Port 			string
	StorageFile		string
}

type StorageManager struct {
	Path			string
}

func (storageManager *StorageManager) BuildStorage() {
	err := os.MkdirAll(storageManager.Path, os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating StorageManager directory.", err)
	}

	file, err := os.Create(storageManager.Path + "/" + INFO_FILE)
	if err != nil {
		log.Fatalf("Error creating StorageManager file.", err)
	}

	file.Close()
}

func (storageManager *StorageManager) UpdateStorage(line string) {
	file, err := os.OpenFile(storageManager.Path + "/" + INFO_FILE, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
        log.Fatalf("Error opening StorageManager file.", err)
    }

    defer file.Close()
 
    _, err = file.WriteString(line)
    if err != nil {
        log.Fatalf("Error writing StorageManager file.", err)
    }

    log.Infof("New message stored in server: %s", line)
}

func (storageManager *StorageManager) GenerateBackup() (string, *os.File) {
	GenerateBackupFile(BACKUP_FILE, storageManager.Path)

	file, err := os.Open(BACKUP_FILE)
	if err != nil {
        log.Fatalf("Error opening compressed backup file.", err)
    }

    hasher := sha256.New()
    if _, err := io.Copy(hasher, file); err != nil {
        log.Fatalf("Error building hash for compressed backup file.", err)
    }

    return fmt.Sprintf("%x", hasher.Sum(nil)), file
}
