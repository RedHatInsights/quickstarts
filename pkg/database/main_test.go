package database

import (
	"fmt"
	"os"
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
	godotenv.Load()
	config.Init()
	cfg := config.Get()
	cfg.Test = true
	time := time.Now().UnixNano()
	dbName = fmt.Sprintf("%d-services.db", time)
	config.Get().DbName = dbName

	Init()
	err := DB.AutoMigrate(&models.Quickstart{}, &models.QuickstartProgress{}, &models.Tag{}, &models.HelpTopic{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Migration complete")
	SeedTags()
}

func tearDown() {
	os.Remove(dbName)
}
