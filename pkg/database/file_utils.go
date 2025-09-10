package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

type contentWrapper struct {
	APIVersion interface{}            `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind       interface{}            `json:"kind,omitempty" yaml:"kind,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       interface{}            `json:"spec,omitempty" yaml:"spec,omitempty"`
	Content    interface{}            `json:"content,omitempty" yaml:"content,omitempty"`
}

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
// Handles various file structures with fallback logic for robustness
func (h *FileHelper) AddTagsToContent(path string, tags interface{}) ([]byte, error) {
	h.logger.Debugf("Adding tags to content: %s", path)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var wrap contentWrapper
	if err := yaml.Unmarshal(raw, &wrap); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	if wrap.Metadata == nil {
		wrap.Metadata = make(map[string]interface{})
	}
	wrap.Metadata["tags"] = tags

	out, err := json.Marshal(wrap)
	if err != nil {
		return nil, fmt.Errorf("marshal %s: %w", path, err)
	}
	h.logger.Debugf("Successfully added tags to content: %s", path)
	return out, nil
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
