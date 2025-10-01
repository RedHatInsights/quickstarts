package database

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/ghodss/yaml"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var migrationTestDB *gorm.DB
var migrationTestDBName string

func setupMigrationTest() {
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
	time := time.Now().UnixNano()
	migrationTestDBName = fmt.Sprintf("%d-migration-test.db", time)
	config.Get().DbName = migrationTestDBName

	// Initialize fresh database for migration test
	Init()
	migrationTestDB = DB
	err = migrationTestDB.AutoMigrate(&models.Quickstart{}, &models.QuickstartProgress{}, &models.Tag{}, &models.HelpTopic{})
	if err != nil {
		panic(err)
	}

	// Run seeding to populate database
	if err := SeedData(); err != nil {
		panic(fmt.Sprintf("Seeding failed in migration test setup: %v", err))
	}
}

func teardownMigrationTest() {
	os.Remove(migrationTestDBName)
}

func TestPostMigrationQuickstartCount(t *testing.T) {
	setupMigrationTest()
	defer teardownMigrationTest()

	// Count expected quickstarts from filesystem
	path, err := os.Getwd()
	assert.NoError(t, err)
	path = strings.TrimSuffix(path, "pkg")

	quickstartsFiles, err := filepath.Glob(path + "/docs/quickstarts/**/metadata.y*")
	assert.NoError(t, err)

	expectedQuickstartsCount := 0
	for _, file := range quickstartsFiles {
		var template MetadataTemplate
		yamlfile, err := ioutil.ReadFile(file)
		if err != nil {
			t.Logf("Warning: Failed to read metadata file %s: %v", file, err)
			continue
		}

		err = yaml.Unmarshal(yamlfile, &template)
		if err != nil {
			t.Logf("Warning: Failed to unmarshal metadata file %s: %v", file, err)
			continue
		}

		if template.Kind == "QuickStarts" {
			expectedQuickstartsCount++
		}
	}

	// Count actual quickstarts in database
	var actualQuickstarts []models.Quickstart
	migrationTestDB.Find(&actualQuickstarts)
	actualQuickstartsCount := len(actualQuickstarts)

	t.Logf("Expected quickstarts from filesystem: %d", expectedQuickstartsCount)
	t.Logf("Actual quickstarts in database: %d", actualQuickstartsCount)

	// Verify counts match
	assert.Equal(t, expectedQuickstartsCount, actualQuickstartsCount,
		"Number of quickstarts in database should match number of QuickStart metadata files")

	// Additional verification: ensure no quickstarts are empty/invalid
	for _, quickstart := range actualQuickstarts {
		assert.NotEmpty(t, quickstart.Name, "Quickstart should have a non-empty name")
		assert.NotEmpty(t, quickstart.Content, "Quickstart should have non-empty content")
	}
}
