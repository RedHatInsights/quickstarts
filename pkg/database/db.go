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
var isFuzzySearchSupported bool

// IsFuzzySearchSupported returns true if the database supports Levenshtein fuzzy search
func IsFuzzySearchSupported() bool {
	return isFuzzySearchSupported
}

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

	// Enable fuzzystrmatch extension for Levenshtein distance fuzzy search (postgres only)
	if cfg.Test {
		logrus.Info("Using SQLite - fuzzy search will fall back to ILIKE")
		isFuzzySearchSupported = false
	} else {
		if err := DB.Exec("CREATE EXTENSION IF NOT EXISTS fuzzystrmatch").Error; err != nil {
			logrus.Warnf("Failed to enable fuzzystrmatch extension: %s", err.Error())
			isFuzzySearchSupported = false
		} else {
			logrus.Info("Fuzzystrmatch extension enabled for fuzzy search")
			isFuzzySearchSupported = true
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

	logrus.Infoln("Database connection established")
}
