package database

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	setUp()
	retCode := m.Run()
	tearDown()
	os.Exit(retCode)
}

var dbName string

func setUp() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
	godotenv.Load()
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

	Init()
	err = DB.AutoMigrate(&models.Tag{}, &models.Quickstart{}, &models.QuickstartProgress{}, &models.HelpTopic{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Migration complete")

	// Ensure clean state for PostgreSQL (SQLite creates a fresh file each run)
	if err := CleanTestTables(); err != nil {
		panic(fmt.Sprintf("CleanTestTables failed: %s", err.Error()))
	}

	SeedTags()
}

func tearDown() {
	if dbName != "" {
		os.Remove(dbName)
	}
}
