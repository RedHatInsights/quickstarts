package utils

import (
	"net/http"
	"net/url"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
)

// ParseLegacyArrayParam extracts array values from legacy bracket notation parameters
// Supports both "param=value" and "param[]=value" formats
func ParseLegacyArrayParam(query url.Values, paramName string) []string {
	// Check legacy bracket notation first
	if values := query[paramName+"[]"]; len(values) > 0 {
		return values
	}

	// Fall back to standard notation
	if values := query[paramName]; len(values) > 0 {
		return values
	}

	return nil
}

// ParseLegacySingleParam extracts single values from legacy bracket notation parameters
// Returns the first value if it's an array parameter
func ParseLegacySingleParam(query url.Values, paramName string) string {
	// Check legacy bracket notation first
	if values := query[paramName+"[]"]; len(values) > 0 && len(values[0]) > 0 {
		return values[0]
	}

	// Fall back to standard notation
	if value := query.Get(paramName); value != "" {
		return value
	}

	return ""
}

// helper for slice‐based params
func setArrayParam[T any](
	query url.Values,
	key string,
	conv func([]string) T,
) *T {
	if vals := ParseLegacyArrayParam(query, key); len(vals) > 0 {
		v := conv(vals)
		return &v
	}
	return nil
}

// helper for single‐value params
func setSingleParam[T any](
	query url.Values,
	key string,
	conv func(string) T,
) *T {
	if val := ParseLegacySingleParam(query, key); val != "" {
		v := conv(val)
		return &v
	}
	return nil
}

// ParseLegacyQuickstartParams handles legacy parameter parsing for quickstart endpoints
func ParseLegacyQuickstartParams(r *http.Request, params *generated.GetQuickstartsParams) {
	q := r.URL.Query()

	params.ProductFamilies = setArrayParam(q, "product-families", func(s []string) generated.ProductFamilies { return generated.ProductFamilies(s) })
	params.Bundle = setArrayParam(q, "bundle", func(s []string) generated.Bundle { return generated.Bundle(s) })
	params.Application = setArrayParam(q, "application", func(s []string) generated.Application { return generated.Application(s) })
	params.Content = setArrayParam(q, "content", func(s []string) generated.Content { return generated.Content(s) })
	params.UseCase = setArrayParam(q, "use-case", func(s []string) generated.UseCase { return generated.UseCase(s) })
	params.Kind = setArrayParam(q, "kind", func(s []string) generated.Kind { return generated.Kind(s) })
	params.Topic = setArrayParam(q, "topic", func(s []string) generated.Topic { return generated.Topic(s) })

	params.Name = setSingleParam(q, "name", func(s string) generated.QuickstartName { return generated.QuickstartName(s) })
	params.DisplayName = setSingleParam(q, "display-name", func(s string) generated.DisplayName { return generated.DisplayName(s) })
}