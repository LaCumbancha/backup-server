package common

import (
	"os"
	"fmt"
	"io/ioutil"
	"crypto/sha256"
	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
)

type BackupStorage struct {
	path		string	
}

type BackupInformation struct {
	Backups 	map[string]BackupRegister 	`yaml:backups`
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
}

func (bkpStorage *BackupStorage) BuildBackupStructure() {
	err := os.MkdirAll(bkpStorage.path, os.ModePerm)
	if err != nil {
		log.Fatalf("Error creating Backups directory.", err)
	}

	file, err := os.Create(bkpStorage.path + "/Information")
	if err != nil {
		log.Fatalf("Error creating BackupInformation file.", err)
	}

	file.Close()
}

func (bkpStorage *BackupStorage) AddBackupClient(backupRegister BackupRegister) string {
	// Read file content
	content, err := ioutil.ReadFile(bkpStorage.path + "/Information")
    if err != nil {
        log.Fatalf("Error reading backups information file.", err)
    }

    // Unmarshall YAML file
    backups := BackupInformation{}
    err = yaml.Unmarshal(content, &backups)
    if err != nil {
		log.Fatalf("Error creating YAML for backups information file.", err)
	}

	if backups.Backups == nil {
		backups.Backups = make(map[string]BackupRegister)
	}

	backupRegisterId := AsSha256(backupRegister)

	if _, ok := backups.Backups[backupRegisterId]; ok {
		log.Infof("Trying to add a backup client with ID %s that was already registered.", backupRegisterId)
		return "Couldn't add new backup client because it already was registered.\n"
	}
		
	backups.Backups[backupRegisterId] = backupRegister

	yamlOutput, err := yaml.Marshal(&backups)
	if err != nil {
		log.Fatalf("Error creating YAML for backups information file.", err)
	}

	err = ioutil.WriteFile(bkpStorage.path + "/Information", yamlOutput, 0644)
	if err != nil {
		log.Fatalf("Error creating backups information file.", err)
	}

	log.Infof("New backup client added with: IP %s; Port %s; Path \"%s\"; Frequency %s. Registered with ID: %s.", backupRegister.Ip, backupRegister.Port, backupRegister.Path, backupRegister.Freq, backupRegisterId)
	return "New backup client request successfully added.\n"
}

func (bkpStorage *BackupStorage) RemoveBackupClient(backupUnregister BackupRegister) string {
	// Read file content
	content, err := ioutil.ReadFile(bkpStorage.path + "/Information")
    if err != nil {
        log.Fatalf("Error reading backups information file.", err)
    }

    // Unmarshall YAML file
    backups := BackupInformation{}
    err = yaml.Unmarshal(content, &backups)
    if err != nil {
		log.Fatalf("Error creating YAML for backups information file.", err)
	}

	backupUnregisterId := AsSha256(backupUnregister)

	if _, ok := backups.Backups[backupUnregisterId]; !ok {
		log.Infof("Trying to remove a backup client with ID %s that was not registered.", backupUnregisterId)
		return "Couldn't remove the backup client because it was not registered.\n"
	}

	delete(backups.Backups, backupUnregisterId)

	yamlOutput, err := yaml.Marshal(&backups)
	if err != nil {
		log.Fatalf("Error creating YAML for backups information file.", err)
	}

	err = ioutil.WriteFile(bkpStorage.path + "/Information", yamlOutput, 0644)
	if err != nil {
		log.Fatalf("Error creating backups information file.", err)
	}

	log.Infof("Removed backup client with ID: %s (IP %s; Port %s; Path \"%s\"; Frequency %s).", backupUnregisterId, backupUnregister.Ip, backupUnregister.Port, backupUnregister.Path, backupUnregister.Freq)
	return "New backup client request successfully removed.\n"
}

func AsSha256(backupRegister BackupRegister) string {
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%v-%v-%v", backupRegister.Ip, backupRegister.Port, backupRegister.Path)))
	return fmt.Sprintf("%x", hasher.Sum(nil))[:7]
}