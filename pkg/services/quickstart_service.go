package services

import (
	"strings"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
)

// QuickstartService handles business logic for quickstarts
type QuickstartService struct{}

// NewQuickstartService creates a new quickstart service
func NewQuickstartService() *QuickstartService {
	return &QuickstartService{}
}

// FindById finds a quickstart by ID
func (s *QuickstartService) FindById(id int) (models.Quickstart, error) {
	var quickStart models.Quickstart
	err := database.DB.First(&quickStart, id).Error
	return quickStart, err
}

// FindByDisplayName finds quickstarts by display name with pagination
func (s *QuickstartService) FindByDisplayName(displayName string, limit, offset int) ([]models.Quickstart, error) {
	var quickStarts []models.Quickstart
	query := database.DB.Offset(offset).Where("content->'spec'->>'displayName' ILIKE ?", "%"+displayName+"%")

	// Apply limit only if it's not -1 (which means no limit)
	if limit != -1 {
		query = query.Limit(limit)
	}

	err := query.Find(&quickStarts).Error
	return quickStarts, err
}

// FindByTagsAndDisplayName finds quickstarts by tags and display name with pagination
func (s *QuickstartService) FindByTagsAndDisplayName(
	tagTypes []models.TagType,
	tagValues [][]string,
	displayName string,
	limit, offset int,
) ([]models.Quickstart, error) {
	var quickstarts []models.Quickstart

	// build "(t.type = ? AND t.value IN (?)) OR …" and collect params
	conds := make([]string, len(tagTypes))
	params := make([]interface{}, 0, len(tagTypes)*2)
	for i, tt := range tagTypes {
		conds[i] = "(t.type = ? AND t.value IN (?))"
		params = append(params, tt, tagValues[i])
	}
	whereClause := strings.Join(conds, " OR ")

	query := database.DB.
		Model(&models.Quickstart{}).
		Joins("JOIN quickstart_tags qt ON qt.quickstart_id = quickstarts.id").
		Joins("JOIN tags t ON t.id = qt.tag_id").
		Where(whereClause, params...).
		Group("quickstarts.id").
		Having("COUNT(DISTINCT t.type) = ?", len(tagTypes))

	if displayName != "" {
		query = query.
			Where("content->'spec'->>'displayName' ILIKE ?", "%"+displayName+"%")
	}
	query = query.Offset(offset)
	if limit != -1 {
		query = query.Limit(limit)
	}

	return quickstarts, query.Find(&quickstarts).Error
}

// Find finds quickstarts based on various criteria
func (s *QuickstartService) Find(tagTypes []models.TagType, tagValues [][]string, name string, displayName string, limit, offset int) ([]models.Quickstart, error) {
	var quickstarts []models.Quickstart
	var err error

	if name != "" {
		err = database.DB.Where("name = ?", name).Find(&quickstarts).Error
	} else if len(tagTypes) > 0 {
		quickstarts, err = s.FindByTagsAndDisplayName(tagTypes, tagValues, displayName, limit, offset)
	} else if displayName != "" {
		quickstarts, err = s.FindByDisplayName(displayName, limit, offset)
	} else {
		query := database.DB.Offset(offset)
		if limit != -1 {
			query = query.Limit(limit)
		}
		err = query.Find(&quickstarts).Error
	}

	return quickstarts, err
}
