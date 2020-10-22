package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/LaCumbancha/backup-server/backup-manager/common"
)

// InitConfig Function that uses viper library to parse env variables. If
// some of the variables cannot be parsed, an error is returned
func InitConfig() (*viper.Viper, error) {
	configEnv := viper.New()

	// Configure viper to read env variables with the BKPMNGR_ prefix
	configEnv.AutomaticEnv()
	configEnv.SetEnvPrefix("bkpmngr")

	// Add env variables supported
	configEnv.BindEnv("port")
	configEnv.BindEnv("storage")
	configEnv.BindEnv("config", "file")

	// Read config file if it's present
	if configFileName := configEnv.GetString("config_file"); configFileName != "" {
		path, file, ctype := GetConfigFile(configFileName)

		configFile = viper.New()
		configFile.SetConfigName(file)
		configFile.SetConfigType(ctype)
		configFile.AddConfigPath(path)
		err := v.ReadInConfig()

		if configFile != nil {
			return nil, errors.Wrapf(err, fmt.Sprintf("Couldn't load config file"))
		}
	}

	return configEnv, configFile, nil
}

func GetConfigFile(configFileName string) (string, string, string) {
	path := filepath.Dir(configFileName)
	file := filepath.Base(configFileName)
	ctype := filepath.Ext(configFileName)[1:]

	return path, file, ctype
}

func main() {
	configEnv, configFile, err := InitConfig()

	if err != nil {
		log.Fatalf("%s", err)
	}

	port := configEnv.GetString("port") || configFile.GetString("port")
	
	if port == "" {
		log.Fatalf("Port variable missing")
	}

	storage := configEnv.GetString("storage") || configFile.GetString("storage")
	
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
