package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("=== Checking if OpenAPI JSON is up to date with YAML source ===")

	// Check if spec/openapi.yaml exists
	if _, err := os.Stat("spec/openapi.yaml"); os.IsNotExist(err) {
		fmt.Println("❌ ERROR: spec/openapi.yaml not found")
		os.Exit(1)
	}

	// Store current state of openapi.json if it exists
	backupCreated := false
	if _, err := os.Stat("spec/openapi.json"); err == nil {
		if err := exec.Command("cp", "spec/openapi.json", "spec/openapi.json.backup").Run(); err != nil {
			fmt.Printf("Warning: Could not backup openapi.json: %v\n", err)
		} else {
			backupCreated = true
		}
	}

	// Ensure backup is cleaned up on exit
	defer func() {
		if backupCreated {
			os.Remove("spec/openapi.json.backup")
		}
	}()

	// Generate JSON from YAML
	fmt.Println("Running 'make openapi-json' to generate JSON from YAML...")
	cmd := exec.Command("make", "openapi-json")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("❌ ERROR: Failed to run 'make openapi-json': %v\n", err)
		fmt.Printf("Output: %s\n", output)
		os.Exit(1)
	}

	// Check if there are any changes to the JSON file
	gitDiffCmd := exec.Command("git", "diff", "--no-index", "--exit-code", "spec/openapi.json", "spec/openapi.json.backup")
	if err := gitDiffCmd.Run(); err != nil {
		// git diff exits with non-zero when there are differences
		fmt.Println("")
		fmt.Println("❌ FAILURE: OpenAPI JSON is NOT up to date with YAML source!")
		fmt.Println("")
		fmt.Println("The spec/openapi.json file is out of sync with spec/openapi.yaml.")
		fmt.Println("Please run 'make openapi-json' locally and commit the updated JSON file.")
		fmt.Println("")
		fmt.Println("Changes detected:")

		// Show the diff
		diffCmd := exec.Command("git", "diff", "spec/openapi.json")
		diffOutput, _ := diffCmd.CombinedOutput()
		fmt.Print(string(diffOutput))

		os.Exit(1)
	}

	fmt.Println("✅ SUCCESS: OpenAPI JSON is up to date with YAML source")
}
