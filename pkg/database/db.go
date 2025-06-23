package database

import (
	"fmt"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	var err error
	var dia gorm.Dialector

	cfg := config.Get()

	var dbdns string
	if cfg.Test {
		dia = sqlite.Open(cfg.DbName)
	} else {
		dbdns = fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=%v", cfg.DbHost, cfg.DbUser, cfg.DbPassword, cfg.DbName, cfg.DbPort, cfg.DbSSLMode)
		if cfg.DbSSLRootCert != "" {
			dbdns = fmt.Sprintf("%s  sslrootcert=%s", dbdns, cfg.DbSSLRootCert)
		}

		dia = postgres.Open(dbdns)
	}

	DB, err = gorm.Open(dia, &gorm.Config{})

	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s", err.Error()))
	}

	logrus.Infoln("Database connection established")

	// Create fuzzystrmatch extension for Levenshtein distance support (PostgreSQL only)
	if !cfg.Test {
		err = DB.Exec("CREATE EXTENSION IF NOT EXISTS fuzzystrmatch").Error
		if err != nil {
			logrus.Warnf("Failed to create fuzzystrmatch extension: %v (Levenshtein search will fall back to ILIKE)", err)
		} else {
			logrus.Infoln("fuzzystrmatch extension available for advanced search")
		}
	}

	if !DB.Migrator().HasTable(&models.Quickstart{}) {
		DB.Migrator().CreateTable(&models.Quickstart{})
	}
	if !DB.Migrator().HasTable(&models.Tag{}) {
		DB.Migrator().CreateTable(&models.Tag{})
	}
	if !DB.Migrator().HasTable(&models.HelpTopic{}) {
		DB.Migrator().CreateTable(&models.HelpTopic{})
	}
	if !DB.Migrator().HasTable(&models.FavoriteQuickstart{}) {
		DB.Migrator().CreateTable(&models.FavoriteQuickstart{})
	}
}
