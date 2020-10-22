package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/LaCumbancha/backup-server/backup-manager/common"
)

func GetConfigFile(configFileName string) (string, string, string) {
	path := filepath.Dir(configFileName)
	file := filepath.Base(configFileName)
	ctype := filepath.Ext(configFileName)[1:]

	return path, file, ctype
}

func InitConfig() (*viper.Viper, *viper.Viper, error) {
	configEnv := viper.New()

	// Configure viper to read env variables with the BKPMNGR_ prefix
	configEnv.AutomaticEnv()
	configEnv.SetEnvPrefix("bkpmngr")

	// Add env variables supported
	configEnv.BindEnv("port")
	configEnv.BindEnv("storage")
	configEnv.BindEnv("config", "file")

	// Read config file if it's present
	var configFile = viper.New()
	if configFileName := configEnv.GetString("config_file"); configFileName != "" {
		path, file, ctype := GetConfigFile(configFileName)

		configFile.SetConfigName(file)
		configFile.SetConfigType(ctype)
		configFile.AddConfigPath(path)
		err := configFile.ReadInConfig()

		if err != nil {
			return nil, nil, errors.Wrapf(err, fmt.Sprintf("Couldn't load config file"))
		}
	}

	return configEnv, configFile, nil
}

// Give precedence to environment variables over configuration file's
func GetConfigValue(configEnv *viper.Viper, configFile *viper.Viper, key string) (string) {
	value := configEnv.GetString(key)
	if value == "" {
		value = configFile.GetString(key)
	}

	return value
}

func main() {
	configEnv, configFile, err := InitConfig()

	if err != nil {
		log.Fatalf("%s", err)
	}

	port := GetConfigValue(configEnv, configFile, "port")
	
	if port == "" {
		log.Fatalf("Port variable missing")
	}

	storage := GetConfigValue(configEnv, configFile, "storage")
	
	if storage == "" {
		log.Fatalf("Storage variable missing")
	}

	managerConfig := common.BackupManagerConfig {
		Port: 		port,
		Storage: 	storage,
	}

	backupManager := common.NewBackupManager(managerConfig)
	backupManager.Run()
}
