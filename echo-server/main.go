package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	log "github.com/sirupsen/logrus"

	"github.com/LaCumbancha/backup-server/echo-server/backup"
	"github.com/LaCumbancha/backup-server/echo-server/utils"
	"github.com/LaCumbancha/backup-server/echo-server/server"
	"github.com/LaCumbancha/backup-server/echo-server/common"
)

func InitConfig() (*viper.Viper, *viper.Viper, error) {
	configEnv := viper.New()

	// Configure viper to read env variables with the BKPMNGR_ prefix
	configEnv.AutomaticEnv()
	configEnv.SetEnvPrefix("app")

	// Add env variables supported
	configEnv.BindEnv("echo", "port")
	configEnv.BindEnv("backup", "port")
	configEnv.BindEnv("storage", "path")
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
	log.SetLevel(log.DebugLevel)
	configEnv, configFile, err := InitConfig()

	if err != nil {
		log.Fatalf("%s", err)
	}

	echoPort := utils.GetConfigValue(configEnv, configFile, "echo_port")
	
	if echoPort == "" {
		log.Fatalf("EchoPort variable missing")
	}

	backupPort := utils.GetConfigValue(configEnv, configFile, "backup_port")
	
	if backupPort == "" {
		log.Fatalf("BackupPort variable missing")
	}

	storage := utils.GetConfigValue(configEnv, configFile, "storage_path")
	
	if storage == "" {
		log.Fatalf("StoragePath variable missing")
	}

	backupServerConfig := common.ServerConfig {
		Port: 			backupPort,
		StoragePath:	storage,
	}

	backupServer := backup.NewBackupServer(backupServerConfig)
	go backupServer.Run()

	echoServerConfig := common.ServerConfig {
		Port: 			echoPort,
		StoragePath:	storage,
	}

	echoServer := server.NewEchoServer(echoServerConfig)
	echoServer.Run()
}
