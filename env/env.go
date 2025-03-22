package env

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

type config struct {
	Server server
	Redis  source
}

type server struct {
	ENV                         string
	Port                        int
	HandlerTimeout              int
	InactiveRouteHandlerTimeout int
	Name                        string
	LogLevel                    string
}

type source struct {
	Host string
	Port int
}

var Conf *config

const (
	EnvVarENV = "Env"
)

func Load() {
	appEnv := os.Getenv(EnvVarENV)

	// set 'local' as default env
	if appEnv == "" {
		appEnv = "local"
	}

	viper.Set(EnvVarENV, appEnv)

	viper.SetConfigName(viper.GetString(EnvVarENV)) // name of config file (without extension)
	viper.AddConfigPath("env/config")               // path to look for the config file in

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatalf("fatal error while reading config file: %v", err)
	}

	err = viper.Unmarshal(&Conf)
	if err != nil {
		log.Fatalf("unable to unmarshal config into struct: %v", err)
	}
}
