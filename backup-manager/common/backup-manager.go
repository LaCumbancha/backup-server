package common

import (
	"io"
	"os"
	"fmt"
	"net"
	"bufio"
	"strings"

	"io/ioutil"
	"encoding/json"
	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
)


type BackupInformation struct {
	Backups 	[]BackupClient 		`yaml:backups`
}

type BackupClient struct {
	Ip 			string 				`yaml:"ip",omitempty`
	Port 		string 				`yaml:"port",omitempty`
	Path 		string 				`yaml:"path",omitempty`
	Freq 		string 				`yaml:"freq",omitempty`
}

type BackupManagerConfig struct {
	Port 		string
	Storage	string
}

type BackupManager struct {
	config 		BackupManagerConfig
	conns   	chan net.Conn
}

func NewBackupManager(config BackupManagerConfig) *BackupManager {
	BackupManager := &BackupManager {
		config: config,
	}

	go BackupManager.buildBackupStructure()
	return BackupManager
}

func ParseAddress(address string) (string, string) {
	split := strings.Split(address, ":")
	ip := split[0]
	port := split[1]

	return ip, port
}

func (bkpManager *BackupManager) buildBackupStructure() {
	os.MkdirAll(bkpManager.config.Storage, os.ModePerm)

	backups := BackupInformation {
		Backups:	[]BackupClient{},
	}

	yamlOutput, err := yaml.Marshal(&backups)
	if err != nil {
		log.Fatalf("Error creating YAML for backups information file.", err)
	}

	err = ioutil.WriteFile(bkpManager.config.Storage + "/Information", yamlOutput, 0644)
	if err != nil {
		log.Fatalf("Error creating backups information file.", err)
	}
}

// Accepting connections
func (bkpManager *BackupManager) acceptConnections(listener net.Listener) chan net.Conn {
	channel := make(chan net.Conn)

	go func() {
		for {
			client, err := listener.Accept()

			if client == nil || err != nil {
				//log.Errorf("Couldn't accept client", err)
				fmt.Printf("Couldn't accept client")
				continue
			}

			ip, port := ParseAddress(client.RemoteAddr().String())
			log.Infof("Got connection from ('%s', %s).", ip, port)

			channel <- client
			log.Infof("Proceed to accept new connections.")
		}
	}()

	return channel
}

// Saving new backup client
func (bkpManager *BackupManager) handleConnections(client net.Conn) {
	buffer := bufio.NewReader(client)
	writer := bufio.NewWriter(client)

	ip, port := ParseAddress(client.RemoteAddr().String())

	for {
		line, err := buffer.ReadBytes('\n')

		if err == io.EOF {
			log.Infof("Connection ('%s', %s) closed.", ip, port)
			break
		} else if err != nil {
			log.Errorf("Couldn't read line", err)
		}

		strLine := string(line)
		log.Infof("Message received from connection ('%s', %s). Msg: %s", ip, port, strLine)

		var backupClient BackupClient
		json.Unmarshal([]byte(strLine), &backupClient)

		if backupClient.Ip == "" || backupClient.Port == "" || backupClient.Path == "" || backupClient.Freq == "" {
			log.Errorf("Error receiving some backup mandatory fields. IP: %s; Port: %s; Path: '%s'; Frequency: %s.", backupClient.Ip, backupClient.Port, backupClient.Path, backupClient.Freq)

			if _, err := writer.WriteString("Error receiving some backup mandatory fields (IP: %s; Port: %s; Path: '%s'; Frequency: %s). Please retry.\n", backupClient.Ip, backupClient.Port, backupClient.Path, backupClient.Freq); err != nil {
				log.Errorf("Error sending response to client from connection ('%s', %s)", ip, port, err)
			} else {
				writer.Flush()
			}
			
		} else {

			log.Infof("New backup client request, with IP %s, port %s, path %s and frequency %s", backupClient.Ip, backupClient.Port, backupClient.Path, backupClient.Freq)
			bkpManager.updateBackupsInfo(backupClient)

			if _, err := writer.WriteString("New backup client request successfully added.\n"); err != nil {
				log.Errorf("Error sending response to client from connection ('%s', %s)", ip, port, err)
			} else {
				writer.Flush()
			}

		}
	}
}

// Update backup information
func (bkpManager *BackupManager) updateBackupsInfo(backupClient BackupClient) {
	// Read file content
	content, err := ioutil.ReadFile(bkpManager.config.Storage + "/Information")
    if err != nil {
        log.Fatalf("Error reading backups information file.", err)
    }

    // Unmarshall YAML file
    backups := BackupInformation{}
    err = yaml.Unmarshal(content, &backups)
    if err != nil {
		log.Fatalf("Error creating YAML for backups information file.", err)
	}

	newBackups := BackupInformation{ Backups:	append(backups.Backups, backupClient) }
	yamlOutput, err := yaml.Marshal(&newBackups)
	if err != nil {
		log.Fatalf("Error creating YAML for backups information file.", err)
	}

	err = ioutil.WriteFile(bkpManager.config.Storage + "/Information", yamlOutput, 0644)
	if err != nil {
		log.Fatalf("Error creating backups information file.", err)
	}

	log.Infof("New backup client added with: IP %s; Port %s; Path \"%s\"; Frequency %s.", backupClient.Ip, backupClient.Port, backupClient.Path, backupClient.Freq)
}


// Run start listening for client messages
func (bkpManager *BackupManager) Run() {
	// Create BackupManager
	listener, err := net.Listen("tcp", ":" + bkpManager.config.Port)
	if listener == nil || err != nil {
		log.Fatalf("Error creating TCP BackupManager socket at port %s.", bkpManager.config.Port, err)
	}

	// Start processing connections
	bkpManager.conns = bkpManager.acceptConnections(listener)

	// Start parallel messages echo
	for {
		go bkpManager.handleConnections(<-bkpManager.conns)
	}
}
