package utils

import (
	"strings"
	"path/filepath"
	"github.com/spf13/viper"
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

// Fill string with '\0' for packange sending
func FillString(message string, size int) string {
	missingPositions := size - len(message)
	return message + strings.Repeat("|", missingPositions)
}
