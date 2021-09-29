package config

import (
	"os"
	"strconv"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
)

type QuickstartsConfig struct {
	ServerAddr      string
	OpenApiSpecPath string
	DbHost          string
	DbUser          string
	DbPassword      string
	DbPort          int
	DbName          string
}

var config *QuickstartsConfig

func Init() {
	config = &QuickstartsConfig{}
	config.ServerAddr = ":8000"
	config.OpenApiSpecPath = "./spec/openapi.json"
	if clowder.IsClowderEnabled() {
		cfg := clowder.LoadedConfig
		config.DbHost = cfg.Database.Hostname
		config.DbPort = cfg.Database.Port
		config.DbUser = cfg.Database.Username
		config.DbPassword = cfg.Database.Password
		config.DbName = cfg.Database.Name
	} else {
		config.DbUser = os.Getenv("PGSQL_USER")
		config.DbPassword = os.Getenv("PGSQL_PASSWORD")
		config.DbHost = os.Getenv("PGSQL_HOSTNAME")
		port, _ := strconv.Atoi(os.Getenv("PGSQL_PORT"))
		config.DbPort = port
		config.DbName = os.Getenv("PGSQL_DATABASE")
	}
}

// Get returns a quickstarts service configuration
func Get() *QuickstartsConfig {
	return config
}
