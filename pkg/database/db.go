package database

import (
	"fmt"
	"os"

	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	var err error
	var dia gorm.Dialector

	dbUser := os.Getenv("PGSQL_USER")
	dbPassword := os.Getenv("PGSQL_PASSWORD")
	dbHostname := os.Getenv("PGSQL_HOSTNAME")
	dbPort := os.Getenv("PGSQL_PORT")
	dbName := os.Getenv("PGSQL_DATABASE")

	dbdns := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=disable", dbHostname, dbUser, dbPassword, dbName, dbPort)

	dia = postgres.Open(dbdns)
	DB, err = gorm.Open(dia, &gorm.Config{})

	if !DB.Migrator().HasTable(&models.Quickstart{}) {
		DB.Migrator().CreateTable(&models.Quickstart{})
	}

	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %s", err.Error()))
	}

	logrus.Infoln("Database conection established")
}
