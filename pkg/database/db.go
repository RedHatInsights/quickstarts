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
		dbdns = fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=disable", cfg.DbHost, cfg.DbUser, cfg.DbPassword, cfg.DbName, cfg.DbPort)
		dia = postgres.Open(dbdns)
	}

	DB, err = gorm.Open(dia, &gorm.Config{})

	if !DB.Migrator().HasTable(&models.Quickstart{}) {
		DB.Migrator().CreateTable(&models.Quickstart{})
	}

	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s", err.Error()))
	}

	logrus.Infoln("Database conection established")
}
