package common

import (
	"io"
	"os"
	"fmt"
	"crypto/md5"
	"archive/tar"
	"compress/gzip"

	log "github.com/sirupsen/logrus"
)

const NO_ETAG = "."

const LOG_DIR = "./data/logs/"
const LOG_FILE = "Log.info"
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
	err := os.MkdirAll(LOG_DIR, os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating ConnectionLogs directory.", err)
	}

	connectionFile, err := os.Create(LOG_DIR + LOG_FILE)
	if err != nil {
		log.Fatalf("Error creating ConnectionLogs file.", err)
	}

	connectionFile.Close()

	err = os.MkdirAll(storageManager.Path, os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating StorageManager directory.", err)
	}

	file, err := os.Create(storageManager.Path + "/" + INFO_FILE)
	if err != nil {
		log.Fatalf("Error creating StorageManager file.", err)
	}

	file.Close()
}

func (storageManager *StorageManager) UpdateStorage(line, ip, port string) {
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

    connectionFile, err := os.OpenFile(LOG_DIR + LOG_FILE, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
        log.Fatalf("Error opening ConnectionLog file.", err)
    }

    defer connectionFile.Close()
 
    _, err = connectionFile.WriteString(fmt.Sprintf("Connection (%s, %s)\n", ip, port))
    if err != nil {
        log.Errorf("Error writing ConnectionLog file.", err)
    }

    log.Infof("New connection stored in log: (%s, %s)", ip, port)
}

func (storageManager *StorageManager) GenerateBackup(path string) (string, *os.File) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
	  	log.Errorf("Requested path to backup doesn't exist.")
        return NO_ETAG, nil
	}

	GenerateBackupFile(BACKUP_FILE, path)

	file, err := os.Open(BACKUP_FILE)
	if err != nil {
        log.Errorf("Error opening compressed backup file.", err)
        return NO_ETAG, nil
    }

    return storageManager.generateEtag(file), file
}

func (storageManager *StorageManager) generateEtag(backupFile *os.File) string {
    gzipFile, err := gzip.NewReader(backupFile)
    if err != nil {
        log.Errorf("Error reading backup gzip file.", err)
        return NO_ETAG
    }

    hasher := md5.New()
    tarReader := tar.NewReader(gzipFile)
    for {

    	fileHeader, err := tarReader.Next()

    	if err == io.EOF {
    		break
    	} else if err != nil {
    		log.Errorf("Error retreaving inner tar files for backup.", err)
    	} else if fileHeader == nil {
    		continue
    	}

    	if _, err = io.Copy(hasher, tarReader); err != nil {
    	    log.Errorf("Error building hash for compressed backup file.", err)
    	    return NO_ETAG
    	}

    }

    return fmt.Sprintf("%x", hasher.Sum(nil))
}