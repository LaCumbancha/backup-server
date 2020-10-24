package common

import (
	"os"

	log "github.com/sirupsen/logrus"
)

const INFO_FILE = "/Data.info"

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

	file, err := os.Create(storageManager.Path + INFO_FILE)
	if err != nil {
		log.Fatalf("Error creating StorageManager file.", err)
	}

	file.Close()
}

func (storageManager *StorageManager) UpdateStorage(line string) {
	file, err := os.OpenFile(storageManager.Path + INFO_FILE, os.O_WRONLY|os.O_APPEND, 0644)
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

func (storageManager *StorageManager) GenerateBackup() {
	Compress("Backup.tar.gz", storageManager.Path)
}
