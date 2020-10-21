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
	v := viper.New()

	// Configure viper to read env variables with the CLI_ prefix
	v.AutomaticEnv()
	v.SetEnvPrefix("bkpmngr")

	// Add env variables supported
	v.BindEnv("port")
	v.BindEnv("storage")
	v.BindEnv("config", "file")

	// Read config file if it's present
	if configFileName := v.GetString("config_file"); configFileName != "" {
		path, file, ctype := GetConfigFile(configFileName)
		v.SetConfigName(file)
		v.SetConfigType(ctype)
		v.AddConfigPath(path)
		err := v.ReadInConfig()

		if err != nil {
			return nil, errors.Wrapf(err, fmt.Sprintf("Couldn't load config file"))
		}
	}

	return v, nil
}

func GetConfigFile(configFileName string) (string, string, string) {
	path := filepath.Dir(configFileName)
	file := filepath.Base(configFileName)
	ctype := filepath.Ext(configFileName)[1:]

	return path, file, ctype
}

func main() {
	v, err := InitConfig()

	if err != nil {
		log.Fatalf("%s", err)
	}

	port := v.GetString("port")
	
	if port == "" {
		log.Fatalf("Port variable missing")
	}

	storage := v.GetString("storage")
	
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
