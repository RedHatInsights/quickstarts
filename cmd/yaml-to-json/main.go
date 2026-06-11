package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ghodss/yaml"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input.yaml> <output.json>\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	// Read YAML file
	yamlData, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading YAML file: %v\n", err)
		os.Exit(1)
	}

	// Convert YAML to JSON
	jsonData, err := yaml.YAMLToJSON(yamlData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting YAML to JSON: %v\n", err)
		os.Exit(1)
	}

	// Pretty print JSON
	var prettyJSON interface{}
	if err := json.Unmarshal(jsonData, &prettyJSON); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	prettyJSONData, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
		os.Exit(1)
	}

	// Write JSON file
	if err := os.WriteFile(outputFile, prettyJSONData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing JSON file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully converted %s to %s\n", inputFile, outputFile)
}
