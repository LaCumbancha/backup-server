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
	"archive/tar"
	"crypto/sha256"
	"compress/gzip"
	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"

	"github.com/LaCumbancha/backup-server/backup-manager/utils"
)

var sizeUnits = map[int]string{
  	0: "B",
  	1: "kB",
  	2: "MB",
  	3: "GB",
  	4: "TB",
  	5: "PB",
  	6: "EB",
  	7: "ZB",
  	8: "YB",
}

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

	backupLog, err := os.OpenFile(bkpStorage.path + backupId + "/Log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("Error opening Backup Log file for ID %s.", backupId, err)
	}
	defer backupLog.Close()

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

	backupFiles := utils.Filter(files, func(fileInfo os.FileInfo) bool { return len(fileInfo.Name()) >= 7 && fileInfo.Name()[0:6] == "Backup" })
	if len(backupFiles) > 0 {
		// Sorting backup files to use last one.
		sort.Slice(backupFiles, func(idx1, idx2 int) bool { return backupFiles[idx1].Name() < backupFiles[idx2].Name() })
		lastBackupName := backupFiles[len(backupFiles) - 1].Name()

		lastBackupFile, err := os.Open(bkpStorage.path + backupId + "/" + lastBackupName)
		if err != nil {
    	    log.Errorf("Error opening backup file %s.", lastBackupName, err)
    	    return ""
    	}
    	defer lastBackupFile.Close()

    	gzipFile, err := gzip.NewReader(lastBackupFile)
    	if err != nil {
    	    log.Errorf("Error reading gzip file %s.", lastBackupName, err)
    	    return ""
    	}

    	hasher := md5.New()
    	tarReader := tar.NewReader(gzipFile)
    	for {

    		fileHeader, err := tarReader.Next()

    		if err == io.EOF {
    			break
    		} else if err != nil {
    			log.Errorf("Error retreaving inner tar files in %s.", lastBackupName, err)
    		} else if fileHeader == nil {
    			continue
    		}

    		if _, err = io.Copy(hasher, tarReader); err != nil {
    		    log.Errorf("Error building hash for compressed backup file.", err)
    		    return ""
    		}

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

func (bkpStorage *BackupStorage) UpdateBackupLog(backupId string, fileSize int64) {
	file, err := os.OpenFile(bkpStorage.path + backupId + "/Log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Warnf("Error opening Backup Log file for ID %s.", backupId, err)
	}
	defer file.Close()

	if fileSize >= 0 {
		size, units := bkpStorage.calculateFileSize(float64(fileSize), 0)
		_, err = file.WriteString(fmt.Sprintf("Registered backup with size %6.1f%s @ %s\n", size, units, time.Now().String()))
	} else {
		_, err = file.WriteString(fmt.Sprintf("Registered backup with unknown size (due to an error) @ %s", time.Now().String()))
	}

    if err != nil {
        log.Errorf("Error writing Backup Log file for ID %s.", backupId, err)
    }
}

func (bkpStorage *BackupStorage) calculateFileSize(size float64, level int) (float64, string) {
	if level == 8 || size < 1024 {
		return size, sizeUnits[level]
	} else {
		return bkpStorage.calculateFileSize(size/1024, level+1)
	}
}

func (bkpStorage *BackupStorage) RetrieveBackupLog(backupRegister BackupRegister) (*os.File, int64){
	backupId := AsSha256(backupRegister)

	file, err := os.OpenFile(bkpStorage.path + backupId + "/Log", os.O_RDONLY, 0644)
	if err != nil {
		log.Errorf("Error opening Backup Log file for ID %s.", backupId, err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		log.Errorf("Error getting backup log stats for ID %s.", backupId, err)
		return file, -1
	}

	return file, fileInfo.Size()
}
