package routes

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
)

func TestMain(m *testing.M) {
	setUp()
	retCode := m.Run()
	tearDown()
	os.Exit(retCode)
}

var dbName string

func setUp() {
	config.Init()
	cfg := config.Get()
	cfg.Test = true

	if testDBURL := os.Getenv("TEST_DATABASE_URL"); testDBURL != "" {
		cfg.TestDatabaseURL = testDBURL
	} else {
		time := time.Now().UnixNano()
		dbName = fmt.Sprintf("%d-services.db", time)
		cfg.DbName = dbName
	}

	database.Init()
	err := database.DB.AutoMigrate(&models.Tag{}, &models.Quickstart{}, &models.QuickstartProgress{}, &models.HelpTopic{}, &models.FavoriteQuickstart{})
	if err != nil {
		panic(err)
	}

	// Ensure clean state for PostgreSQL (SQLite creates a fresh file each run)
	if err := database.CleanTestTables(); err != nil {
		panic(fmt.Sprintf("CleanTestTables failed: %s", err.Error()))
	}
}

func tearDown() {
	if dbName != "" {
		os.Remove(dbName)
	}
}
