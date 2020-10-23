package main

import (
	"fmt"
	"log"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/LaCumbancha/backup-server/backup-manager/utils"
	"github.com/LaCumbancha/backup-server/backup-manager/common"
)

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
		path, file, ctype := utils.GetConfigFile(configFileName)

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

func main() {
	configEnv, configFile, err := InitConfig()

	if err != nil {
		log.Fatalf("%s", err)
	}

	port := utils.GetConfigValue(configEnv, configFile, "port")
	
	if port == "" {
		log.Fatalf("Port variable missing")
	}

	storage := utils.GetConfigValue(configEnv, configFile, "storage")
	
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
