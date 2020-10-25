package common

import (
	"os"
	"fmt"
	"time"
	"sync"
	"io/ioutil"
	"crypto/sha256"
	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
)

type BackupStorageConfig struct {
	Path 			string
}

type BackupStorage struct {
	path		string
	mutex 		sync.Mutex	
}

type BackupRequest struct {
	Verb		string
	Args		BackupRegister
}

type BackupRegister struct {
	Ip 			string 						`yaml:"ip",omitempty`
	Port 		string 						`yaml:"port",omitempty`
	Path 		string 						`yaml:"path",omitempty`
	Freq 		string 						`yaml:"freq",omitempty`
	Next		time.Time 					`yaml:"next",omitempty`
}

func NewBackupStorage(config BackupStorageConfig) *BackupStorage {
	path := config.Path

	if path[len(path) - 1] != '/' {
		log.Debugf("Adding '/' at the end of backup path.")
		path += "/"
	}

	backupStorage := &BackupStorage {
		path: 		path,
	}

	return backupStorage
}

func (bkpStorage *BackupStorage) BuildBackupStructure() {
	err := os.MkdirAll(bkpStorage.path, os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating Backups directory.", err)
	}

	file, err := os.OpenFile(bkpStorage.path + BACKUP_INFORMATION, os.O_RDONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating BackupInformation file.", err)
	}

	file.Close()
}

func (bkpStorage *BackupStorage) readBackupInformation() map[string]BackupRegister {
	// Read file content
	content, err := ioutil.ReadFile(bkpStorage.path + BACKUP_INFORMATION)
    if err != nil {
        log.Fatalf("Error reading backups information file.", err)
    }

    // Unmarshall YAML file
    var backups map[string]BackupRegister
    err = yaml.Unmarshal(content, &backups)
    if err != nil {
		log.Fatalf("Error creating YAML for backups information file.", err)
	}

	return backups
}

func (bkpStorage *BackupStorage) writeBackupInformation(backups map[string]BackupRegister) {
	// Generate YAML file.
	yamlOutput, err := yaml.Marshal(&backups)
	if err != nil {
		log.Fatalf("Error updating YAML for backups information file.", err)
	}

	// Write YAML file.
	err = ioutil.WriteFile(bkpStorage.path + BACKUP_INFORMATION, yamlOutput, 0644)
	if err != nil {
		log.Fatalf("Error updating backups information file.", err)
	}
}

func (bkpStorage *BackupStorage) GetBackupClients() map[string]BackupRegister {
	bkpStorage.mutex.Lock()
	backups := bkpStorage.readBackupInformation()
	bkpStorage.mutex.Unlock()
	return backups
}

func (bkpStorage *BackupStorage) UpdateBackupClients(updatedBackups map[string]BackupRegister) {
	bkpStorage.mutex.Lock()
	backups := bkpStorage.readBackupInformation()

	// Updating backup values
	for updatedBackupId, updatedBackupInfo := range updatedBackups {

		if _, ok := backups[updatedBackupId]; !ok {
			log.Infof("Trying to update backup client with ID %s, but it was unregistered.", updatedBackupId)
		} else {
			backups[updatedBackupId] = updatedBackupInfo
			log.Debugf("Updating BackupInfo for client with ID %s.", updatedBackupId)
		}

	}

	bkpStorage.writeBackupInformation(backups)
	bkpStorage.mutex.Unlock()
}

func (bkpStorage *BackupStorage) AddBackupClient(backupRegister BackupRegister) string {
	backupRegisterId := AsSha256(backupRegister)

	// Update next backup information
	freqDuration, err := time.ParseDuration(backupRegister.Freq)
	if err != nil {
        log.Infof("Invalid frequency format given: %s (client: %s).", backupRegister.Freq, backupRegisterId, err)
        return "Coudln't register new backup client. Invalid frequency format.\n"
    }

	backupRegister.Next = time.Now().Add(freqDuration)

	bkpStorage.mutex.Lock()
	backups := bkpStorage.readBackupInformation()

	if backups == nil {
		backups = make(map[string]BackupRegister)
	}

	if _, ok := backups[backupRegisterId]; ok {
		log.Infof("Trying to add a backup client with ID %s that was already registered.", backupRegisterId)
		return "Couldn't add new backup client because it already was registered.\n"
	}
		
	backups[backupRegisterId] = backupRegister

	bkpStorage.writeBackupInformation(backups)
	bkpStorage.mutex.Unlock()

	err = os.Mkdir(bkpStorage.path + backupRegisterId, os.ModePerm)
	if err != nil {
		log.Errorf("Error creating Backup directory for id %s.", backupRegisterId, err)
		return "Couldn't add new backup client because it already was registered.\n"
	}

	log.Infof("New backup client added for id with: IP %s; Port %s; Path \"%s\"; Frequency %s. Registered with ID: %s.", backupRegister.Ip, backupRegister.Port, backupRegister.Path, backupRegister.Freq, backupRegisterId)
	return "New backup client request successfully added.\n"
}

func (bkpStorage *BackupStorage) RemoveBackupClient(backupUnregister BackupRegister) string {
	backupUnregisterId := AsSha256(backupUnregister)

	bkpStorage.mutex.Lock()
	backups := bkpStorage.readBackupInformation()

	if _, ok := backups[backupUnregisterId]; !ok {
		log.Infof("Trying to remove a backup client with ID %s that was not registered.", backupUnregisterId)
		return "Couldn't remove the backup client because it was not registered.\n"
	}

	delete(backups, backupUnregisterId)

	bkpStorage.writeBackupInformation(backups)
	bkpStorage.mutex.Unlock()

	log.Infof("Removed backup client with ID: %s (IP %s; Port %s; Path \"%s\"; Frequency %s).", backupUnregisterId, backupUnregister.Ip, backupUnregister.Port, backupUnregister.Path, backupUnregister.Freq)
	return "New backup client request successfully removed.\n"
}

func AsSha256(backupRegister BackupRegister) string {
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%v-%v-%v", backupRegister.Ip, backupRegister.Port, backupRegister.Path)))
	return fmt.Sprintf("%x", hasher.Sum(nil))[:11]
}
