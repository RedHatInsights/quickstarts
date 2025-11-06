package database

import (
	"database/sql"
	"fmt"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Register custom Levenshtein function for SQLite
func init() {
	sql.Register("sqlite3_custom",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("levenshtein", levenshtein, true)
			},
		})
}

func Init() {
	var err error
	var dia gorm.Dialector

	cfg := config.Get()

	var dbdns string
	if cfg.Test {
		// Use custom SQLite driver with Levenshtein function
		dia = sqlite.Dialector{
			DriverName: "sqlite3_custom",
			DSN:        cfg.DbName,
		}
	} else {
		dbdns = fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=%v", cfg.DbHost, cfg.DbUser, cfg.DbPassword, cfg.DbName, cfg.DbPort, cfg.DbSSLMode)
		if cfg.DbSSLRootCert != "" {
			dbdns = fmt.Sprintf("%s  sslrootcert=%s", dbdns, cfg.DbSSLRootCert)
		}

		dia = postgres.Open(dbdns)
	}

	DB, err = gorm.Open(dia, &gorm.Config{})

	// Enable fuzzystrmatch extension for Levenshtein distance fuzzy search
	if !cfg.Test { // Only enable for non-test environments (PostgreSQL)
		DB.Exec("CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;")
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

	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s", err.Error()))
	}

	logrus.Infoln("Database connection established")
}
