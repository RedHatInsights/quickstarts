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

	logrus.Info("Starting database migration")
	err := database.DB.AutoMigrate(&models.Quickstart{}, &models.QuickstartProgress{}, &models.Tag{}, &models.HelpTopic{}, &models.FavoriteQuickstart{})
	if err != nil {
		logrus.Fatalf("Database migration failed: %v", err)
	}
	logrus.Info("Database migration completed successfully")

	logrus.Info("Starting database seeding")
	if err := database.SeedData(); err != nil {
		logrus.Fatalf("Database seeding failed: %v", err)
		panic("Quickstarts db seeding process failure, do not panic!")
	}
	logrus.Info("Database seeding completed successfully")
}
