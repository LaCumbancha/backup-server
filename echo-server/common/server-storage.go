package common

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

type EchoStorage struct {
	file		string
}

func (echoStorage *EchoStorage) BuildStorageFile() {
	err := os.MkdirAll(filepath.Dir(echoStorage.file), os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating EchoStorage directory.", err)
	}

	file, err := os.Create(echoStorage.file)
	if err != nil {
		log.Fatalf("Error creating EchoStorage file.", err)
	}

	file.Close()
}

func (echoStorage *EchoStorage) UpdateStorage(line string) {
	file, err := os.OpenFile(echoStorage.file, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
        log.Fatalf("Error opening EchoStorage file.", err)
    }

    defer file.Close()
 
    _, err = file.WriteString(line)
    if err != nil {
        log.Fatalf("Error writing EchoStorage file.", err)
    }

    log.Infof("New message stored in server: %s", line)
}
