package main

import (
	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	godotenv.Load()
	config.Init()
	database.Init()
	err := database.DB.AutoMigrate(&models.Quickstart{})
	if err != nil {
		panic(err)
	}
	logrus.Info("Migration complete")
}
