package common

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

type StorageConfig struct {
	Port 			string
	StorageFile		string
}

type StorageManager struct {
	file		string
}

func (storageManager *StorageManager) BuildStorageFile() {
	err := os.MkdirAll(filepath.Dir(storageManager.file), os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating StorageManager directory.", err)
	}

	file, err := os.Create(storageManager.file)
	if err != nil {
		log.Fatalf("Error creating StorageManager file.", err)
	}

	file.Close()
}

func (storageManager *StorageManager) UpdateStorage(line string) {
	file, err := os.OpenFile(storageManager.file, os.O_WRONLY|os.O_APPEND, 0644)
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
	file, err := os.OpenFile(storageManager.file, os.O_WRONLY|os.O_APPEND, 0644)
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
