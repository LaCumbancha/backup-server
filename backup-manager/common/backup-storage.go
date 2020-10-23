package common

import (
	"os"
	"io/ioutil"
	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
)

type BackupStorage struct {
	path		string	
}

type BackupInformation struct {
	Backups 	[]BackupClient 		`yaml:backups`
}

type BackupClient struct {
	Ip 			string 				`yaml:"ip",omitempty`
	Port 		string 				`yaml:"port",omitempty`
	Path 		string 				`yaml:"path",omitempty`
	Freq 		string 				`yaml:"freq",omitempty`
}

func (bkpStorage *BackupStorage) BuildBackupStructure() {
	os.MkdirAll(bkpStorage.path, os.ModePerm)

	backups := BackupInformation {
		Backups:	[]BackupClient{},
	}

	yamlOutput, err := yaml.Marshal(&backups)
	if err != nil {
		log.Fatalf("Error creating YAML for backups information file.", err)
	}

	err = ioutil.WriteFile(bkpStorage.path + "/Information", yamlOutput, 0644)
	if err != nil {
		log.Fatalf("Error creating backups information file.", err)
	}
}

// Update backup information
func (bkpStorage *BackupStorage) UpdateBackupInfo(backupRequest BackupClient) {
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

	newBackups := BackupInformation{ Backups: append(backups.Backups, backupRequest) }
	yamlOutput, err := yaml.Marshal(&newBackups)
	if err != nil {
		log.Fatalf("Error creating YAML for backups information file.", err)
	}

	err = ioutil.WriteFile(bkpStorage.path + "/Information", yamlOutput, 0644)
	if err != nil {
		log.Fatalf("Error creating backups information file.", err)
	}

	log.Infof("New backup client added with: IP %s; Port %s; Path \"%s\"; Frequency %s.", backupRequest.Ip, backupRequest.Port, backupRequest.Path, backupRequest.Freq)
}
