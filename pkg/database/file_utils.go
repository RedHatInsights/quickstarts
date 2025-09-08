package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

// FileHelper provides consistent file operations with error handling and logging
type FileHelper struct {
	logger *logrus.Entry
}

// NewFileHelper creates a new file helper with consistent logging
func NewFileHelper(context string) *FileHelper {
	return &FileHelper{
		logger: logrus.WithField("context", context),
	}
}

// ReadYAMLFile reads and parses a YAML file into the provided interface
func (h *FileHelper) ReadYAMLFile(filePath string, dest interface{}) error {
	h.logger.Debugf("Reading YAML file: %s", filePath)
	
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		h.logger.Errorf("Failed to read file %s: %v", filePath, err)
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	
	if err := yaml.Unmarshal(data, dest); err != nil {
		h.logger.Errorf("Failed to unmarshal YAML from %s: %v", filePath, err)
		return fmt.Errorf("failed to unmarshal YAML from %s: %w", filePath, err)
	}
	
	h.logger.Debugf("Successfully read YAML file: %s", filePath)
	return nil
}

// ReadJSONFromYAML reads a YAML file and converts it to JSON bytes
func (h *FileHelper) ReadJSONFromYAML(filePath string) ([]byte, error) {
	h.logger.Debugf("Reading and converting YAML to JSON: %s", filePath)
	
	yamlData, err := ioutil.ReadFile(filePath)
	if err != nil {
		h.logger.Errorf("Failed to read YAML file %s: %v", filePath, err)
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	
	jsonData, err := yaml.YAMLToJSON(yamlData)
	if err != nil {
		h.logger.Errorf("Failed to convert YAML to JSON for %s: %v", filePath, err)
		return nil, fmt.Errorf("failed to convert YAML to JSON for %s: %w", filePath, err)
	}
	
	h.logger.Debugf("Successfully converted YAML to JSON: %s", filePath)
	return jsonData, nil
}

// GlobFiles finds files matching a pattern with error handling and logging
func (h *FileHelper) GlobFiles(pattern string) ([]string, error) {
	h.logger.Debugf("Searching for files with pattern: %s", pattern)
	
	files, err := filepath.Glob(pattern)
	if err != nil {
		h.logger.Errorf("Failed to glob files with pattern %s: %v", pattern, err)
		return nil, fmt.Errorf("failed to glob files with pattern %s: %w", pattern, err)
	}
	
	h.logger.Infof("Found %d files matching pattern %s", len(files), pattern)
	return files, nil
}

// AddTagsToContent reads a content file, adds tags to its metadata, and returns JSON
func (h *FileHelper) AddTagsToContent(contentPath string, tags interface{}) ([]byte, error) {
	h.logger.Debugf("Adding tags to content file: %s", contentPath)
	
	jsonContent, err := h.ReadJSONFromYAML(contentPath)
	if err != nil {
		return nil, err
	}
	
	// Parse as generic interface{} to handle different file structures
	var data interface{}
	if err := json.Unmarshal(jsonContent, &data); err != nil {
		h.logger.Errorf("Failed to unmarshal JSON content for %s: %v", contentPath, err)
		return nil, fmt.Errorf("failed to unmarshal JSON content for %s: %w", contentPath, err)
	}
	
	// Convert to map for manipulation
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("content file %s does not have expected top-level object structure", contentPath)
	}
	
	// Ensure metadata exists and is a map
	if dataMap["metadata"] == nil {
		dataMap["metadata"] = make(map[string]interface{})
	}
	
	metadata, ok := dataMap["metadata"].(map[string]interface{})
	if !ok {
		dataMap["metadata"] = make(map[string]interface{})
		metadata = dataMap["metadata"].(map[string]interface{})
	}
	
	// Add tags to metadata
	metadata["tags"] = tags
	
	result, err := json.Marshal(dataMap)
	if err != nil {
		h.logger.Errorf("Failed to marshal JSON content for %s: %v", contentPath, err)
		return nil, fmt.Errorf("failed to marshal JSON content for %s: %w", contentPath, err)
	}
	
	h.logger.Debugf("Successfully added tags to content file: %s", contentPath)
	return result, nil
}

// ExtractStringFromMetadata extracts a string field from JSON metadata
func (h *FileHelper) ExtractStringFromMetadata(jsonData []byte, field string, filePath string) (string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return "", fmt.Errorf("failed to unmarshal content for %s: %w", filePath, err)
	}
	
	metadata, ok := data["metadata"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("metadata section not found or invalid in %s", filePath)
	}
	
	fieldInterface, exists := metadata[field]
	if !exists {
		return "", fmt.Errorf("%s not found in metadata for %s", field, filePath)
	}
	
	fieldStr, ok := fieldInterface.(string)
	if !ok {
		return "", fmt.Errorf("%s is not a string in metadata for %s", field, filePath)
	}
	
	if fieldStr == "" {
		return "", fmt.Errorf("%s is empty in metadata for %s", field, filePath)
	}
	
	return fieldStr, nil
}