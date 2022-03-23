package database

import (
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestCreateTags(t *testing.T) {
	t.Run("create TAG with corrent tag type", func(t *testing.T) {
		var tag models.Tag
		tag.Type = models.ApplicationTag
		tag.Value = "foo"
		error := DB.Save(&tag).Error
		assert.Equal(t, nil, error)

		var allTags []models.Tag
		var newTag models.Tag
		DB.Find(&allTags)
		assert.Equal(t, 1, len(allTags))
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

		var quickStarts []models.Quickstart
		var quickStartsAssociations []models.Quickstart
		var dbTag models.Tag
		DB.Find(&dbTag, tag.ID)
		DB.Find(&quickStarts)
		DB.Model(&tag).Association("Quickstarts").Find(&quickStartsAssociations)
		assert.Equal(t, dbTag.ID, tag.ID)
		assert.Equal(t, 1, len(quickStarts))
		assert.Equal(t, 1, len(quickStartsAssociations))
		assert.Equal(t, "baz", quickStartsAssociations[0].Name)
		assert.Equal(t, quickStart.ID, quickStartsAssociations[0].ID)
	})
}
