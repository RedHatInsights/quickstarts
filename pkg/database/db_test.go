package database

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
)

func TestCreateTags(t *testing.T) {
	t.Run("create TAG with correct tag type", func(t *testing.T) {
		var tag models.Tag
		var allTags []models.Tag
		DB.Find(&allTags)
		var initLen = len(allTags)
		tag.Type = models.ApplicationTag
		tag.Value = "foo"
		error := DB.Save(&tag).Error
		assert.Equal(t, nil, error)

		var newTag models.Tag
		DB.Find(&allTags)
		assert.Equal(t, initLen+1, len(allTags))
		DB.Find(&newTag, tag.ID)
		assert.Equal(t, models.ApplicationTag, newTag.Type)
		assert.Equal(t, "foo", newTag.Value)
	})

	t.Run("fail to create tag with invalid tag type", func(t *testing.T) {
		var tag models.Tag
		tag.Type = "nonsense"
		tag.Value = "foo"
		error := DB.Create(&tag).Error
		// Error message differs between SQLite and PostgreSQL drivers,
		// but the Go-level Valuer always returns "invalid tag value".
		assert.Error(t, error)
		assert.Contains(t, error.Error(), "invalid tag value")
	})

	t.Run("fail to create tag with empty tag type", func(t *testing.T) {
		var tag models.Tag
		tag.Value = "foo"
		error := DB.Create(&tag).Error
		assert.Error(t, error)
		assert.Contains(t, error.Error(), "invalid tag value")
	})

	t.Run("fail to create tag with empty tag value", func(t *testing.T) {
		var tag models.Tag
		tag.Type = models.BundleTag
		error := DB.Create(&tag).Error
		// SQLite: "NOT NULL constraint failed: tags.value"
		// PostgreSQL: 'null value in column "value" ... violates not-null constraint'
		assert.Error(t, error)
	})
}

func TestCreateQuickstartWithBundle(t *testing.T) {
	t.Run("create quickstart with a rhel bundle tag", func(t *testing.T) {
		var quickStart models.Quickstart
		var tag models.Tag
		var error error

		/**quickstart creating should be fine*/
		quickStart.Content = []byte(`{"foo": "bar"}`)
		quickStart.Name = "baz"
		error = DB.Create(&quickStart).Error
		assert.Equal(t, nil, error)

		/**Tag creating should be fine*/
		tag.Type = models.BundleTag
		tag.Value = "rhel"
		error = DB.Create(&tag).Error
		assert.Equal(t, nil, error)
		DB.Model(&tag).Association("Quickstarts").Append(&quickStart)
		error = DB.Save(&tag).Error
		assert.Equal(t, nil, error)

		quickstartFiles, _ := filepath.Glob(filepath.Join(contentDir(), "quickstarts", "*", "metadata.y*"))
		t.Log(quickstartFiles)
		quickstart_len := len(quickstartFiles)
		t.Log(quickstart_len)
		var quickStarts []models.Quickstart
		var quickStartsAssociations []models.Quickstart
		var dbTag models.Tag
		DB.Find(&dbTag, tag.ID)
		DB.Find(&quickStarts)
		DB.Model(&tag).Association("Quickstarts").Find(&quickStartsAssociations)
		assert.Equal(t, dbTag.ID, tag.ID)
		assert.Equal(t, quickstart_len+1, len(quickStarts))
		assert.Equal(t, 1, len(quickStartsAssociations))
		assert.Equal(t, "baz", quickStartsAssociations[0].Name)
		assert.Equal(t, quickStart.ID, quickStartsAssociations[0].ID)
	})
}

func TestClearOldContentWithFavorites(t *testing.T) {
	t.Run("seeding succeeds with pre-existing favorites", func(t *testing.T) {
		// Insert a quickstart so we can create a favorite referencing it.
		qs := models.Quickstart{Name: "test-favorite-qs", Content: []byte(`{"metadata":{"name":"test-favorite-qs"}}`)}
		assert.NoError(t, DB.Create(&qs).Error)

		fav := models.FavoriteQuickstart{
			AccountId:      "test-account",
			QuickstartName: "test-favorite-qs",
			Favorite:       true,
		}
		assert.NoError(t, DB.Create(&fav).Error)

		// Verify favorite exists before seeding.
		var count int64
		DB.Model(&models.FavoriteQuickstart{}).Count(&count)
		assert.Greater(t, count, int64(0), "favorites should exist before re-seed")

		// Re-seed — this previously crashed with "unsupported relations: Quickstart".
		SeedTags()

		// Seeding should complete without panic/error.
		// Favorites are cleared then restored for quickstarts that still exist.
		var quickstarts []models.Quickstart
		DB.Find(&quickstarts)
		assert.Greater(t, len(quickstarts), 0, "quickstarts should exist after seeding")
	})
}

func TestIdempotentReseeding(t *testing.T) {
	t.Run("running SeedTags twice produces consistent state", func(t *testing.T) {
		SeedTags()

		var firstQuickstarts []models.Quickstart
		var firstHelpTopics []models.HelpTopic
		var firstTags []models.Tag
		DB.Find(&firstQuickstarts)
		DB.Find(&firstHelpTopics)
		DB.Find(&firstTags)

		SeedTags()

		var secondQuickstarts []models.Quickstart
		var secondHelpTopics []models.HelpTopic
		var secondTags []models.Tag
		DB.Find(&secondQuickstarts)
		DB.Find(&secondHelpTopics)
		DB.Find(&secondTags)

		assert.Equal(t, len(firstQuickstarts), len(secondQuickstarts), "quickstart count should be stable across re-seeds")
		assert.Equal(t, len(firstHelpTopics), len(secondHelpTopics), "help topic count should be stable across re-seeds")
		assert.Equal(t, len(firstTags), len(secondTags), "tag count should be stable across re-seeds")
	})
}

func TestDBSeeding(t *testing.T) {
	base := contentDir()
	quickstartsFiles, err := filepath.Glob(filepath.Join(base, "quickstarts", "*", "metadata.y*"))
	if err != nil {
		log.Fatal(err)
	}
	helpTopicsFiles, err := filepath.Glob(filepath.Join(base, "help-topics", "*", "metadata.y*"))
	if err != nil {
		log.Fatal(err)
	}
	files := append(quickstartsFiles, helpTopicsFiles...)
	t.Log(files)

	t.Run("create DB seeding", func(t *testing.T) {
		var quickStarts []models.Quickstart
		DB.Find(&quickStarts)
	})

	t.Run("DB contains correct quickstart data", func(t *testing.T) {
		var metadataTemplates []MetadataTemplate
		metadataTemplates = findTags()

		for _, template := range metadataTemplates {
			if template.Kind == "QuickStarts" {
				var quickstart models.Quickstart
				yamlfile, err := ioutil.ReadFile(template.ContentPath)
				if err != nil {
					t.Log(err)
				}
				jsonContent, err := yaml.YAMLToJSON(yamlfile)
				var data map[string]map[string]string
				json.Unmarshal(jsonContent, &data)
				name := data["metadata"]["name"]
				DB.Where("name = ?", name).Find(&quickstart)
				var db_data map[string]map[string]string
				json.Unmarshal([]byte(quickstart.Content), &db_data)
				assert.Equal(t, db_data["metadata"]["name"], name)
				assert.Equal(t, db_data["metadata"]["content"], data["metadata"]["content"])
			}
		}
	})
	t.Run("DB contains correct help topic data", func(t *testing.T) {
		var metadataTemplates []MetadataTemplate
		metadataTemplates = findTags()

		for _, template := range metadataTemplates {
			if template.Kind == "HelpTopic" {
				yamlfile, err := ioutil.ReadFile(template.ContentPath)
				if err != nil {
					t.Log(err)
				}
				jsonContent, err := yaml.YAMLToJSON(yamlfile)
				var data []map[string]interface{}
				json.Unmarshal(jsonContent, &data)
				for _, d := range data {
					var helptopic models.HelpTopic
					name := d["name"]
					DB.Where("name = ?", name).Find(&helptopic)
					content := d["content"]
					var db_data map[string]interface{}
					json.Unmarshal([]byte(helptopic.Content), &db_data)
					assert.Equal(t, db_data["content"], content)
					assert.Equal(t, db_data["name"], d["name"])
				}
			}
		}
	})
}
