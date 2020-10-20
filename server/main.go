package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/LaCumbancha/backup-server/server/common"
)

// InitConfig Function that uses viper library to parse env variables. If
// some of the variables cannot be parsed, an error is returned
func InitConfig() (*viper.Viper, error) {
	v := viper.New()

	// Configure viper to read env variables with the CLI_ prefix
	v.AutomaticEnv()
	v.SetEnvPrefix("server")

	// Add env variables supported
	v.BindEnv("admin", "port")
	v.BindEnv("backup", "path")
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

	port := v.GetString("admin_port")
	
	if port == "" {
		log.Fatalf("Port variable missing")
	}

	backupPath := v.GetString("backup_path")
	
	if backupPath == "" {
		log.Fatalf("BackupPath variable missing")
	}

	serverConfig := common.ServerConfig {
		Port: 		port,
		BackupPath: backupPath,
	}

	server := common.NewServer(serverConfig)
	server.Run()
}
