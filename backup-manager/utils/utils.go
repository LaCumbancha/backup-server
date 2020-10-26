package utils

import (
	"net"
	"bufio"
	"strings"
	"path/filepath"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"

	"github.com/LaCumbancha/backup-server/backup-manager/common"
)

// Get configuration file's path structure. 
func GetConfigFile(configFileName string) (string, string, string) {
	path := filepath.Dir(configFileName)
	file := filepath.Base(configFileName)
	ctype := filepath.Ext(configFileName)[1:]

	return path, file, ctype
}

// Give precedence to environment variables over configuration file's
func GetConfigValue(configEnv *viper.Viper, configFile *viper.Viper, key string) (string) {
	value := configEnv.GetString(key)
	if value == "" {
		value = configFile.GetString(key)
	}

	return value
}

// Parse address into IP and port
func ParseAddress(address string) (string, string) {
	split := strings.Split(address, ":")
	ip := split[0]
	port := split[1]

	return ip, port
}

// Write to socket
func SocketWrite(message string, socket net.Conn) {
	writer := bufio.NewWriter(socket)
	ip, port := ParseAddress(socket.RemoteAddr().String())

	if _, err := writer.WriteString(message); err != nil {
		log.Errorf("Error sending message to client from connection ('%s', %s). Message: %s", ip, port, message, err)
	} else {
		writer.Flush()
	}
}

// Fill string with '|' for packange sending
func FillString(message string, size int) string {
	missingPositions := size - len(message)
	return message + strings.Repeat(common.PADDING_CHARACTER, missingPositions)
}

// Remove padding '|'
func UnfillString(message []byte) string {
	tmpStr := reversed(string(message))

	trimmed := ""
	for _, char := range tmpStr {
		str := string(char)
		if str != common.PADDING_CHARACTER {
			trimmed += str
		}
	}

	return reversed(trimmed)
}

func reversed(str string) string {
	result := ""
	for _, char := range str { 
        result = string(char) + result 
    }
    return result
}
