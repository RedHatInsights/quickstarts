package services

import (
	"strings"

	"github.com/RedHatInsights/quickstarts/config"
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

// FindByDisplayNameFuzzy finds quickstarts by display name using Levenshtein distance for fuzzy matching
// Falls back to ILIKE search if Levenshtein is not available (e.g., SQLite) or if no fuzzy results found
// Uses word-level matching for single-word queries to handle typos like "ansibel" → "Ansible"
func (s *QuickstartService) FindByDisplayNameFuzzy(searchTerm string, limit, offset int) ([]models.Quickstart, error) {
	var quickstarts []models.Quickstart

	// Check if fuzzy search is supported (PostgreSQL with fuzzystrmatch extension)
	if !database.IsFuzzySearchSupported() {
		// Fall back to regular ILIKE search for SQLite or if extension is not available
		return s.FindByDisplayName(searchTerm, limit, offset)
	}

	cfg := config.Get()
	threshold := cfg.MaxFuzzySearchDistance

	// Smart detection: Use word-level matching for single-word queries
	isSingleWord := !strings.Contains(strings.TrimSpace(searchTerm), " ")

	var err error

	if isSingleWord {
		// Word-level fuzzy matching: Split display name into words and match against each
		// This handles "ansibel" → "ansible" in "Create your first Ansible Playbook"

		// Build the SQL query conditionally based on whether we have a limit
		sqlQuery := `
			WITH word_distances AS (
				SELECT
					q.*,
					MIN(levenshtein(LOWER(?), word)) as distance
				FROM quickstarts q,
				LATERAL unnest(regexp_split_to_array(LOWER(q.content->'spec'->>'displayName'), '\s+')) as word
				WHERE q.content->'spec'->>'displayName' IS NOT NULL
				GROUP BY q.id, q.created_at, q.updated_at, q.deleted_at, q.name, q.content
				HAVING MIN(levenshtein(LOWER(?), word)) <= ?
			)
			SELECT * FROM word_distances
			ORDER BY distance ASC, content->'spec'->>'displayName' ASC`

		if limit == -1 {
			// No limit - just add offset
			sqlQuery += ` OFFSET ?`
			err = database.DB.Raw(sqlQuery, searchTerm, searchTerm, threshold, offset).Find(&quickstarts).Error
		} else {
			// With limit
			sqlQuery += ` LIMIT ? OFFSET ?`
			err = database.DB.Raw(sqlQuery, searchTerm, searchTerm, threshold, limit, offset).Find(&quickstarts).Error
		}
	} else {
		// Full-phrase fuzzy matching: Compare entire search term to entire display name
		// This handles "Getting started with automation hb" → "...hub"
		query := database.DB.
			Select("quickstarts.*, levenshtein(LOWER(content->'spec'->>'displayName'), LOWER(?)) as distance", searchTerm).
			Where("levenshtein(LOWER(content->'spec'->>'displayName'), LOWER(?)) <= ?", searchTerm, threshold).
			Order("distance ASC, content->'spec'->>'displayName' ASC").
			Offset(offset)

		if limit != -1 {
			query = query.Limit(limit)
		}

		err = query.Find(&quickstarts).Error
	}
	if err != nil {
		return quickstarts, err
	}

	// Hybrid fallback: If no fuzzy results found, fall back to ILIKE for partial matching
	// This handles cases where the search term is much shorter than the display name
	// (e.g., "ansible" vs "Create your first Ansible Playbook")
	if len(quickstarts) == 0 {
		return s.FindByDisplayName(searchTerm, limit, offset)
	}

	return quickstarts, nil
}

// FindByTagsAndDisplayNameFuzzy finds quickstarts by tags and display name using fuzzy matching
// Falls back to ILIKE search if Levenshtein is not available (e.g., SQLite) or if no fuzzy results found
func (s *QuickstartService) FindByTagsAndDisplayNameFuzzy(
	tagTypes []models.TagType,
	tagValues [][]string,
	searchTerm string,
	limit, offset int,
) ([]models.Quickstart, error) {
	var quickstarts []models.Quickstart

	// Check if fuzzy search is supported (PostgreSQL with fuzzystrmatch extension)
	if !database.IsFuzzySearchSupported() {
		// Fall back to regular ILIKE search for SQLite or if extension is not available
		return s.FindByTagsAndDisplayName(tagTypes, tagValues, searchTerm, limit, offset)
	}

	cfg := config.Get()
	threshold := cfg.MaxFuzzySearchDistance

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
		Select("quickstarts.*, levenshtein(LOWER(content->'spec'->>'displayName'), LOWER(?)) as distance", searchTerm).
		Joins("JOIN quickstart_tags qt ON qt.quickstart_id = quickstarts.id").
		Joins("JOIN tags t ON t.id = qt.tag_id").
		Where(whereClause, params...).
		Group("quickstarts.id").
		Having("COUNT(DISTINCT t.type) = ?", len(tagTypes))

	if searchTerm != "" {
		query = query.
			Where("levenshtein(LOWER(content->'spec'->>'displayName'), LOWER(?)) <= ?", searchTerm, threshold).
			Order("distance ASC, content->'spec'->>'displayName' ASC")
	}

	query = query.Offset(offset)
	if limit != -1 {
		query = query.Limit(limit)
	}

	err := query.Find(&quickstarts).Error
	if err != nil {
		return quickstarts, err
	}

	// Hybrid fallback: If no fuzzy results found, fall back to ILIKE for partial matching
	if len(quickstarts) == 0 && searchTerm != "" {
		return s.FindByTagsAndDisplayName(tagTypes, tagValues, searchTerm, limit, offset)
	}

	return quickstarts, nil
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

// FindFuzzy finds quickstarts using fuzzy search with Levenshtein distance
func (s *QuickstartService) FindFuzzy(tagTypes []models.TagType, tagValues [][]string, name string, searchTerm string, limit, offset int) ([]models.Quickstart, error) {
	var quickstarts []models.Quickstart
	var err error

	if name != "" {
		// Exact name match takes precedence
		err = database.DB.Where("name = ?", name).Find(&quickstarts).Error
	} else if len(tagTypes) > 0 {
		quickstarts, err = s.FindByTagsAndDisplayNameFuzzy(tagTypes, tagValues, searchTerm, limit, offset)
	} else if searchTerm != "" {
		quickstarts, err = s.FindByDisplayNameFuzzy(searchTerm, limit, offset)
	} else {
		query := database.DB.Offset(offset)
		if limit != -1 {
			query = query.Limit(limit)
		}
		err = query.Find(&quickstarts).Error
	}

	return quickstarts, err
}
