package services

import (
	"fmt"
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

// findFuzzy is a unified fuzzy search implementation that supports optional tag filtering
// Pass nil/empty slices for tagTypes/tagValues when searching without tag filters
func (s *QuickstartService) findFuzzy(
	tagTypes []models.TagType,
	tagValues [][]string,
	searchTerm string,
	limit, offset int,
) ([]models.Quickstart, error) {
	var quickstarts []models.Quickstart

	// Check if fuzzy search is supported (PostgreSQL with fuzzystrmatch extension)
	if !database.IsFuzzySearchSupported() {
		// Fall back to regular ILIKE search
		if len(tagTypes) > 0 {
			return s.FindByTagsAndDisplayName(tagTypes, tagValues, searchTerm, limit, offset)
		}
		return s.FindByDisplayName(searchTerm, limit, offset)
	}

	cfg := config.Get()
	threshold := cfg.MaxFuzzySearchDistance

	// Build the base query with optional tag filtering
	var baseTableQuery string
	var params []interface{}

	// Start with searchTerm (used in query_words CTE)
	params = append(params, searchTerm)

	if len(tagTypes) > 0 {
		// Build tag filter conditions
		conds := make([]string, len(tagTypes))

		// Add tag parameters BEFORE threshold
		for i, tt := range tagTypes {
			conds[i] = "(t.type = ? AND t.value IN (?))"
			params = append(params, tt, tagValues[i])
		}
		whereClause := strings.Join(conds, " OR ")

		// CTE that filters quickstarts by tags first
		baseTableQuery = `
		tagged_quickstarts AS (
			SELECT q.id, q.created_at, q.updated_at, q.deleted_at, q.name, q.content
			FROM quickstarts q
			JOIN quickstart_tags qt ON qt.quickstart_id = q.id
			JOIN tags t ON t.id = qt.tag_id
			WHERE ` + whereClause + `
			GROUP BY q.id, q.created_at, q.updated_at, q.deleted_at, q.name, q.content
			HAVING COUNT(DISTINCT t.type) = ` + fmt.Sprintf("%d", len(tagTypes)) + `
		),`
	} else {
		baseTableQuery = ""
	}

	// Add threshold AFTER tag parameters (used in WHERE min_distance <= ?)
	params = append(params, threshold)

	// Determine which table to use in word_matches CTE
	sourceTable := "quickstarts q"
	sourceAlias := "q"
	if len(tagTypes) > 0 {
		sourceTable = "tagged_quickstarts tq"
		sourceAlias = "tq"
	}

	// Word-by-word fuzzy matching with partial matches:
	// 1. Split query into words
	// 2. For each query word, find the best matching word in each display name
	// 3. Return quickstarts that match at least one query word within threshold
	// 4. Order by: number of matching words (DESC), then total distance (ASC)
	sqlQuery := `
		WITH query_words AS (
			SELECT unnest(regexp_split_to_array(LOWER(?), '\s+')) as query_word
		),
		` + baseTableQuery + `
		word_matches AS (
			SELECT
				` + sourceAlias + `.id,
				` + sourceAlias + `.created_at,
				` + sourceAlias + `.updated_at,
				` + sourceAlias + `.deleted_at,
				` + sourceAlias + `.name,
				` + sourceAlias + `.content,
				qw.query_word,
				MIN(levenshtein(qw.query_word, display_word)) as min_distance
			FROM query_words qw
			CROSS JOIN ` + sourceTable + `
			CROSS JOIN LATERAL unnest(regexp_split_to_array(LOWER(` + sourceAlias + `.content->'spec'->>'displayName'), '\s+')) as display_word
			WHERE ` + sourceAlias + `.content->'spec'->>'displayName' IS NOT NULL
			GROUP BY ` + sourceAlias + `.id, ` + sourceAlias + `.created_at, ` + sourceAlias + `.updated_at, ` + sourceAlias + `.deleted_at, ` + sourceAlias + `.name, ` + sourceAlias + `.content, qw.query_word
		)
		SELECT
			id, created_at, updated_at, deleted_at, name, content,
			COUNT(*) as match_count,
			SUM(min_distance) as total_distance
		FROM word_matches
		WHERE min_distance <= ?
		GROUP BY id, created_at, updated_at, deleted_at, name, content
		ORDER BY match_count DESC, total_distance ASC, content->'spec'->>'displayName' ASC`

	var err error
	if limit == -1 {
		sqlQuery += ` OFFSET ?`
		params = append(params, offset)
	} else {
		sqlQuery += ` LIMIT ? OFFSET ?`
		params = append(params, limit, offset)
	}

	err = database.DB.Raw(sqlQuery, params...).Find(&quickstarts).Error
	if err != nil {
		return quickstarts, err
	}

	// Hybrid fallback: If no fuzzy results found, fall back to ILIKE for partial matching
	if len(quickstarts) == 0 {
		if len(tagTypes) > 0 {
			return s.FindByTagsAndDisplayName(tagTypes, tagValues, searchTerm, limit, offset)
		}
		return s.FindByDisplayName(searchTerm, limit, offset)
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
	// Use fuzzy search when there's a search term or tag filters
	if searchTerm != "" || len(tagTypes) > 0 {
		return s.findFuzzy(tagTypes, tagValues, searchTerm, limit, offset)
	}

	// Otherwise fall back to normal Find (handles exact name match, all quickstarts, etc.)
	return s.Find(tagTypes, tagValues, name, "", limit, offset)
}
