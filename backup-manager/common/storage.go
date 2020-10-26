package common

import (
	"io"
	"os"
	"fmt"
	"time"
	"sync"
	"sort"
	"io/ioutil"
	"crypto/md5"
	"crypto/sha256"
	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
)

const MAX_BACKUPS = 10

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

	bkpStorage.initializeBackupRegister(backupRegisterId)

	log.Infof("New backup client added for ID %s with: IP %s; Port %s; Path \"%s\"; Frequency %s. Registered with ID: %s.", backupRegisterId, backupRegister.Ip, backupRegister.Port, backupRegister.Path, backupRegister.Freq, backupRegisterId)
	return fmt.Sprintf("New backup client successfully added with ID %s.", backupRegisterId)
}

func (bkpStorage *BackupStorage) initializeBackupRegister(backupId string) bool {
	err := os.Mkdir(bkpStorage.path + backupId, os.ModePerm)
	if err != nil {
		log.Errorf("Error creating Backup directory for ID %s.", backupId, err)
		return false
	}

	bkpStorage.updateBackupRegisterHistoric(backupId, "Backup client registered")
    return true
}

func (bkpStorage *BackupStorage) updateBackupRegisterHistoric(backupId, message string) {
	file, err := os.OpenFile(bkpStorage.path + backupId + "/Historic", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("Error opening Backup Historic file for ID %s.", backupId, err)
	}
	defer file.Close()

	_, err = file.WriteString(message + fmt.Sprintf(" at %s.\n", time.Now().String()))
    if err != nil {
        log.Errorf("Error writing Backup Historic file for ID %s.", backupId, err)
    }
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

	bkpStorage.updateBackupRegisterHistoric(backupUnregisterId, "Backup client unregistered")
	log.Infof("Removed backup client with ID: %s (IP %s; Port %s; Path \"%s\"; Frequency %s).", backupUnregisterId, backupUnregister.Ip, backupUnregister.Port, backupUnregister.Path, backupUnregister.Freq)
	return "Backup client successfully removed.\n"
}

func (bkpStorage *BackupStorage) GenerateEtag(backupId string) string {
	files, err := ioutil.ReadDir(bkpStorage.path + backupId)
	if err != nil {
		log.Errorf("Error reading backup directory for client %s", backupId, err)
		bkpStorage.checkForDirectory(backupId)
		return ""
	}

	if len(files) > 0 {
		// Sorting backup files to use last one.
		sort.Slice(files, func(idx1, idx2 int) bool { return files[idx1].Name() < files[idx2].Name() })
		lastBackupName := files[len(files) - 1].Name()

		lastBackupFile, err := os.Open(bkpStorage.path + backupId + "/" + lastBackupName)
		if err != nil {
    	    log.Fatalf("Error opening backup file %s.", lastBackupName, err)
    	}
    	defer lastBackupFile.Close()

		hasher := md5.New()
    	if _, err := io.Copy(hasher, lastBackupFile); err != nil {
    	    log.Errorf("Error building hash for compressed backup file.", err)
    	    return ""
    	}
    	
    	return fmt.Sprintf("%x", hasher.Sum(nil))

	} else {
		log.Debugf("No backup file to calculate Etag, defaulting with empty string.")
		return ""
	}
}

func (bkpStorage *BackupStorage) checkForDirectory(backupId string) bool {
	if _, err := os.Stat(bkpStorage.path + backupId); os.IsNotExist(err) {
		log.Warnf("Backup directory for ID %s was missing.", backupId)
		return bkpStorage.initializeBackupRegister(backupId)
	}
	return true
}

func AsSha256(backupRegister BackupRegister) string {
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%v-%v-%v", backupRegister.Ip, backupRegister.Port, backupRegister.Path)))
	return fmt.Sprintf("%x", hasher.Sum(nil))[:11]
}

func (bkpStorage *BackupStorage) AddNewBackup(backupId string) *os.File {
	oldBackups, err := ioutil.ReadDir(bkpStorage.path + backupId)
	if err != nil {
		log.Errorf("Error reading backup directory for client %s", backupId, err)
		if !bkpStorage.checkForDirectory(backupId) {
			return nil
		}
	}

	newFile, err := os.Create(bkpStorage.path + backupId + "/Backup-" + fmt.Sprintf(time.Now().Format("20060102150405")) + ".tar.gz")
	if err != nil {
		log.Errorf("Error creating new backup received from client %s.", backupId)
		return nil
	}

	bkpStorage.updateBackupRegisterHistoric(backupId, "New backup saved")

	if len(oldBackups) > MAX_BACKUPS {
		sort.Slice(oldBackups, func(idx1, idx2 int) bool { return oldBackups[idx1].Name() < oldBackups[idx2].Name() })
		oldestFile := oldBackups[0].Name()

		os.Remove(oldestFile)
		log.Infof("Max backups capacity reached. Removing oldest file: %s.", oldestFile)
		bkpStorage.updateBackupRegisterHistoric(backupId, fmt.Sprintf("Old backup removed (%s) due to max capacity reached", oldestFile))
	}

	return newFile
}
