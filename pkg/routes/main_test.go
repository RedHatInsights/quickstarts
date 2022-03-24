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
	time := time.Now().UnixNano()
	dbName = fmt.Sprintf("%d-services.db", time)
	config.Get().DbName = dbName

	database.Init()
	err := database.DB.AutoMigrate(&models.Quickstart{}, &models.QuickstartProgress{}, &models.HelpTopic{})
	if err != nil {
		panic(err)
	}
}

func tearDown() {
	os.Remove(dbName)
}
