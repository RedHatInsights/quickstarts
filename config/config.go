package config

import (
	"os"
	"strconv"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/sirupsen/logrus"
)

type QuickstartsConfig struct {
	ServerAddr      string
	OpenApiSpecPath string
	DbHost          string
	DbUser          string
	DbPassword      string
	DbPort          int
	DbName          string
	MetricsPort     int
	Test            bool
	DbSSLMode       string
	DbSSLRootCert   string
	LogLevel        string
}

var config *QuickstartsConfig

func Init() {
	config = &QuickstartsConfig{}
	config.ServerAddr = ":8000"
	config.OpenApiSpecPath = "./spec/openapi.json"
	config.Test = false
	// Log level will default to "Error". Level should be one of
	// info or debug or error
	level, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		level = logrus.ErrorLevel.String()
	}
	config.LogLevel = level
	if clowder.IsClowderEnabled() {
		cfg := clowder.LoadedConfig
		config.DbHost = cfg.Database.Hostname
		config.DbPort = cfg.Database.Port
		config.DbUser = cfg.Database.Username
		config.DbPassword = cfg.Database.Password
		config.DbName = cfg.Database.Name
		config.MetricsPort = cfg.MetricsPort
		config.DbSSLMode = cfg.Database.SslMode
		certPath, err := cfg.RdsCa()
		if err != nil {
			logrus.Info("Failed to load DB cert path")
			config.DbSSLMode = "disable"
			config.DbSSLRootCert = ""
		} else {
			config.DbSSLRootCert = certPath
		}

	} else {
		config.DbUser = os.Getenv("PGSQL_USER")
		config.DbPassword = os.Getenv("PGSQL_PASSWORD")
		config.DbHost = os.Getenv("PGSQL_HOSTNAME")
		port, _ := strconv.Atoi(os.Getenv("PGSQL_PORT"))
		config.DbPort = port
		config.DbName = os.Getenv("PGSQL_DATABASE")
		config.MetricsPort = 8080
		config.DbSSLMode = "disable"
		config.DbSSLRootCert = ""
	}
}

// Get returns a quickstarts service configuration
func Get() *QuickstartsConfig {
	return config
}
