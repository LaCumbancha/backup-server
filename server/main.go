package main

import (
	"log"

	"github.com/spf13/viper"
	"github.com/LaCumbancha/docker-init/server/common"
)

// InitConfig Function that uses viper library to parse env variables. If
// some of the variables cannot be parsed, an error is returned
func InitConfig() (*viper.Viper) {
	v := viper.New()

	// Configure viper to read env variables with the CLI_ prefix
	v.AutomaticEnv()
	v.SetEnvPrefix("server")

	// Add env variables supported
	v.BindEnv("port")

	return v
}

func main() {
	v := InitConfig()

	port := v.GetString("port")
	
	if port == "" {
		log.Fatalf("Port variable missing")
	}

	serverConfig := common.ServerConfig {
		Port: port,
	}

	server := common.NewServer(serverConfig)
	server.Run()
}
