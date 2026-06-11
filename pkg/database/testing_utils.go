package database

import (
	"fmt"

	"github.com/RedHatInsights/quickstarts/config"
)

// CleanTestTables truncates all model tables and resets serial sequences.
// Only runs in test mode against PostgreSQL (SQLite tests use a fresh file each run).
func CleanTestTables() error {
	if DB == nil || !config.Get().Test || DB.Dialector.Name() != "postgres" {
		return nil
	}
	tables := []string{
		"quickstart_tags",
		"help_topic_tags",
		"favorite_quickstarts",
		"quickstart_progresses",
		"tags",
		"help_topics",
		"quickstarts",
	}
	for _, table := range tables {
		if err := DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table)).Error; err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}
	return nil
}
