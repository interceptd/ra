package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	gitingestPath     string
	gitingestOutput   string
	gitingestJSONOnly bool
)

// gitingestCmd represents the gitingest command
var gitingestCmd = &cobra.Command{
	Use:   "gitingest [path]",
	Short: "Run gitingest analysis on a repository",
	Long: `Run gitingest analysis on a repository to generate comprehensive 
repository documentation including summary, tree structure, and content.

This command will run the Python gitingest script and save the results
to timestamped files in the output directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGitingest,
}

func init() {
	rootCmd.AddCommand(gitingestCmd)

	// Gitingest-specific flags
	gitingestCmd.Flags().StringVarP(&gitingestPath, "path", "p", ".", "path to analyze (default: current directory)")
	gitingestCmd.Flags().StringVarP(&gitingestOutput, "output", "o", "output", "output directory for results")
	gitingestCmd.Flags().BoolVar(&gitingestJSONOnly, "json-only", false, "only generate JSON output")

	// Bind flags to viper
	viper.BindPFlag("gitingest.path", gitingestCmd.Flags().Lookup("path"))
	viper.BindPFlag("gitingest.output", gitingestCmd.Flags().Lookup("output"))
	viper.BindPFlag("gitingest.json-only", gitingestCmd.Flags().Lookup("json-only"))
}

func runGitingest(cmd *cobra.Command, args []string) error {
	// Use path from args if provided, otherwise use flag value
	targetPath := gitingestPath
	if len(args) > 0 {
		targetPath = args[0]
	}

	// Check if target path exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("target path does not exist: %s", targetPath)
	}

	// Get absolute path
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Create output directory
	if err := createOutputDir(gitingestOutput); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Check if Python is available
	if err := checkPythonAvailable(); err != nil {
		return fmt.Errorf("Python is not available: %v", err)
	}

	// Check if gitingest is available
	if err := checkGitingestAvailable(); err != nil {
		return fmt.Errorf("gitingest is not available: %v", err)
	}

	// Run gitingest analysis
	fmt.Printf("Running gitingest analysis on: %s\n", absPath)
	fmt.Printf("Output directory: %s\n", gitingestOutput)

	if err := runGitingestAnalysis(absPath); err != nil {
		return fmt.Errorf("failed to run gitingest analysis: %v", err)
	}

	fmt.Println("✅ Gitingest analysis completed successfully!")
	return nil
}

func checkPythonAvailable() error {
	// Check if virtual environment exists
	venvPython := getVenvPython()
	if _, err := os.Stat(venvPython); os.IsNotExist(err) {
		return fmt.Errorf("Python virtual environment not found. Please run: ./repo-analyzer setup")
	}

	// Test virtual environment Python
	cmd := exec.Command(venvPython, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("virtual environment Python is not working. Please run: ./repo-analyzer setup")
	}

	if verbose {
		fmt.Printf("Found Python in virtual environment: %s\n", venvPython)
	}

	return nil
}

func checkGitingestAvailable() error {
	// Check if virtual environment exists
	venvPython := getVenvPython()
	if _, err := os.Stat(venvPython); os.IsNotExist(err) {
		return fmt.Errorf("Python virtual environment not found. Please run: ./repo-analyzer setup")
	}

	// Try to import gitingest in virtual environment
	cmd := exec.Command(venvPython, "-c", "import gitingest; print('gitingest available')")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gitingest is not available in virtual environment. Please run: ./repo-analyzer setup")
	}

	if verbose {
		fmt.Printf("Gitingest is available in virtual environment: %s\n", venvPython)
	}

	return nil
}

func runGitingestAnalysis(targetPath string) error {
	// Create a temporary file for JSON output
	tempFile := filepath.Join(os.TempDir(), "gitingest_results.json")
	defer os.Remove(tempFile)

	// Create a temporary Python script to run gitingest and output JSON
	jsonOnlyStr := "False"
	if gitingestJSONOnly {
		jsonOnlyStr = "True"
	}

	pythonScript := fmt.Sprintf(`
import os
import json
from datetime import datetime
from gitingest import ingest

def main():
    target_path = %q
    json_only = %s
    temp_file = %q
    
    # Check if the target path exists
    if not os.path.exists(target_path):
        print(f"Error: The target path '{target_path}' does not exist.")
        return 1
    
    print(f"Starting gitingest parsing of: {target_path}")
    print("This may take a moment...")
    
    try:
        # Use gitingest to parse the target
        summary, tree, content = ingest(target_path)
        
        # Generate timestamp
        timestamp = datetime.now().strftime("%%Y%%m%%d_%%H%%M%%S")
        
        # Create base name
        base_name = os.path.basename(os.path.abspath(target_path))
        if not base_name:
            base_name = "root"
        
        # Create results dictionary
        results = {
            "timestamp": timestamp,
            "target_path": target_path,
            "base_name": base_name,
            "summary": summary,
            "tree": tree,
            "content": content,
            "json_only": json_only,
            "stats": {
                "summary_lines": len(summary.split('\n')),
                "tree_lines": len(tree.split('\n')),
                "content_lines": len(content.split('\n')),
                "content_size": len(content)
            }
        }
        
        # Write results to temp file
        with open(temp_file, 'w', encoding='utf-8') as f:
            json.dump(results, f, ensure_ascii=False, indent=2)
        
        print("GITINGEST_COMPLETED")
        
    except Exception as e:
        print(f"Error during gitingest analysis: {e}")
        return 1
    
    return 0

if __name__ == "__main__":
    exit(main())
`, targetPath, jsonOnlyStr, tempFile)

	// Use virtual environment Python
	venvPython := getVenvPython()

	// Execute the Python script and capture output
	cmd := exec.Command(venvPython, "-c", pythonScript)
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error: %s\n", stderr.String())
		return err
	}

	// Check if the script completed successfully
	output := stdout.String()
	if !strings.Contains(output, "GITINGEST_COMPLETED") {
		fmt.Print(output)
		return fmt.Errorf("gitingest analysis did not complete successfully")
	}

	// Read the JSON results from the temp file
	jsonData, err := os.ReadFile(tempFile)
	if err != nil {
		return fmt.Errorf("failed to read JSON results file: %v", err)
	}

	// Parse JSON results
	var results map[string]interface{}
	if err := json.Unmarshal(jsonData, &results); err != nil {
		return fmt.Errorf("failed to parse JSON results: %v", err)
	}

	// Extract data from results
	timestamp := results["timestamp"].(string)
	baseName := results["base_name"].(string)
	summary := results["summary"].(string)
	tree := results["tree"].(string)
	content := results["content"].(string)
	jsonOnly := results["json_only"].(bool)
	stats := results["stats"].(map[string]interface{})

	// Create output filenames
	summaryFile := filepath.Join(gitingestOutput, fmt.Sprintf("%s_summary_%s.txt", baseName, timestamp))
	treeFile := filepath.Join(gitingestOutput, fmt.Sprintf("%s_tree_%s.txt", baseName, timestamp))
	contentFile := filepath.Join(gitingestOutput, fmt.Sprintf("%s_content_%s.txt", baseName, timestamp))
	jsonFile := filepath.Join(gitingestOutput, fmt.Sprintf("%s_results_%s.json", baseName, timestamp))

	// Write files using Go's text cleaning functions
	if !jsonOnly {
		// Write summary file with cleaning
		if err := writeCleanFile(summaryFile, summary); err != nil {
			return fmt.Errorf("failed to write summary file: %v", err)
		}
		fmt.Printf("✅ Summary saved to: %s\n", summaryFile)

		// Write tree file with cleaning
		if err := writeCleanFile(treeFile, tree); err != nil {
			return fmt.Errorf("failed to write tree file: %v", err)
		}
		fmt.Printf("✅ Tree structure saved to: %s\n", treeFile)

		// Write content file with cleaning
		if err := writeCleanFile(contentFile, content); err != nil {
			return fmt.Errorf("failed to write content file: %v", err)
		}
		fmt.Printf("✅ Content saved to: %s\n", contentFile)
	}

	// Write JSON file with cleaning
	if err := writeCleanJSONFile(jsonFile, string(jsonData)); err != nil {
		return fmt.Errorf("failed to write JSON file: %v", err)
	}
	fmt.Printf("✅ JSON results saved to: %s\n", jsonFile)

	// Print analysis summary
	fmt.Printf("\n📊 Analysis Summary:\n")
	fmt.Printf("   Target: %s\n", targetPath)
	fmt.Printf("   Summary: %.0f lines\n", stats["summary_lines"])
	fmt.Printf("   Tree: %.0f lines\n", stats["tree_lines"])
	fmt.Printf("   Content: %.0f lines (%.0f bytes)\n", stats["content_lines"], stats["content_size"])

	return nil
}
