package routes

import (
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/RedHatInsights/quickstarts/pkg/utils"
)

// QuickstartsQuery holds everything your service.Find() needs
type QuickstartsQuery struct {
	Name, DisplayName string
	Limit, Offset     int
	TagTypes          []models.TagType
	TagValues         [][]string
}

// NewQuickstartsQuery parses pagination, legacy params and builds tag filters.
// Keeps all the messy bits out of your HTTP handler.
func NewQuickstartsQuery(r *http.Request, p generated.GetQuickstartsParams) QuickstartsQuery {
	utils.ParseLegacyQuickstartParams(r, &p)

	q := QuickstartsQuery{
		Name:        optionalQuickstartName(p.Name),
		DisplayName: optionalDisplayName(p.DisplayName),
		Limit:       sanitizeLimit(utils.ConvertIntPtr(p.Limit, 50)),
		Offset:      sanitizeOffset(utils.ConvertIntPtr(p.Offset, 0)),
	}

	// build a quick map of all tag types → values
	tagMap := map[models.TagType][]string{
		"bundle":           utils.ConvertStringSlice(p.Bundle),
		"application":      utils.ConvertStringSlice(p.Application),
		"product-families": utils.ConvertStringSlice(p.ProductFamilies),
		"use-case":         utils.ConvertStringSlice(p.UseCase),
		"content":          utils.ConvertStringSlice(p.Content),
		"kind":             utils.ConvertStringSlice(p.Kind),
		"topic":            utils.ConvertStringSlice(p.Topic),
	}

	var tagTypeInstance models.TagType
	allTagTypes := tagTypeInstance.GetAllTags()

	for _, tt := range allTagTypes {
		if vals := tagMap[tt]; len(vals) > 0 {
			q.TagTypes = append(q.TagTypes, tt)
			q.TagValues = append(q.TagValues, vals)
		}
	}

	return q
}

func optionalQuickstartName(n *generated.QuickstartName) string {
	if n != nil {
		return string(*n)
	}
	return ""
}

func optionalDisplayName(n *generated.DisplayName) string {
	if n != nil {
		return string(*n)
	}
	return ""
}

func sanitizeLimit(l int) int {
	if l == 0 || l < -1 {
		return 50
	}
	return l
}

func sanitizeOffset(o int) int {
	if o < 0 {
		return 0
	}
	return o
}