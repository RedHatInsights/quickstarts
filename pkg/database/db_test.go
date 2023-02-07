package database

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
)

func TestCreateTags(t *testing.T) {
	t.Run("create TAG with correct tag type", func(t *testing.T) {
		var tag models.Tag
		tag.Type = models.ApplicationTag
		tag.Value = "foo"
		error := DB.Save(&tag).Error
		assert.Equal(t, nil, error)

		var allTags []models.Tag
		var newTag models.Tag
		DB.Find(&allTags)
		assert.Equal(t, 7, len(allTags))
		DB.Find(&newTag, tag.ID)
		assert.Equal(t, models.ApplicationTag, newTag.Type)
		assert.Equal(t, "foo", newTag.Value)
	})

	t.Run("fail to create tag with invalid tag type", func(t *testing.T) {
		var tag models.Tag
		tag.Type = "nonsense"
		tag.Value = "foo"
		error := DB.Create(&tag).Error
		assert.Equal(t, "sql: converting argument $4 type: invalid tag value", error.Error())
	})

	t.Run("fail to create tag with empty tag type", func(t *testing.T) {
		var tag models.Tag
		tag.Value = "foo"
		error := DB.Create(&tag).Error
		assert.Equal(t, "sql: converting argument $4 type: invalid tag value", error.Error())
	})

	t.Run("fail to create tag with empty tag value", func(t *testing.T) {
		var tag models.Tag
		tag.Type = models.BundleTag
		error := DB.Create(&tag).Error
		assert.Equal(t, "NOT NULL constraint failed: tags.value", error.Error())
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

		path, _ := os.Getwd()
		quickstartFiles, _ := filepath.Glob(path + "/docs/quickstarts/**/metadata.y*")
		quickstart_len := len(quickstartFiles)
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

func TestDBSeeding(t *testing.T) {
	path, err := os.Getwd()
	path = strings.TrimRight(path, "pkg")
	quickstartsFiles, err := filepath.Glob(path + "/docs/quickstarts/**/metadata.y*")
	if err != nil {
		log.Fatal(err)
	}
	helpTopicsFiles, err := filepath.Glob(path + "/docs/help-topics/**/metadata.y*")
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
