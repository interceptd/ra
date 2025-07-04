package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	repomixPath            string
	repomixOutput          string
	repomixRemote          string
	repomixIncludePatterns string
	repomixIgnorePatterns  string
	// Output Format Options
	repomixIncludeFileSummary        bool
	repomixIncludeDirectoryStructure bool
	repomixShowLineNumbers           bool
	repomixOutputParsableFormat      bool
	// File Processing Options
	repomixRemoveComments   bool
	repomixRemoveEmptyLines bool
	// Additional options
	repomixTopFilesLength int
	repomixHeaderText     string
)

// repomixCmd represents the repomix command
var repomixCmd = &cobra.Command{
	Use:   "repomix [path]",
	Short: "Run repomix analysis on a repository",
	Long: `Run repomix analysis on a repository to generate AI-friendly 
repository packaging in markdown format with customizable options.

This command will run the npx repomix tool with the specified options
and save the results to timestamped files in the output directory.

Features:
- Markdown output format (no code compression)
- Parsable format optimized for AI consumption
- File summary and directory structure inclusion
- Comment and empty line removal options
- Support for local and remote repositories`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRepomix,
}

func init() {
	rootCmd.AddCommand(repomixCmd)

	// Basic flags
	repomixCmd.Flags().StringVarP(&repomixPath, "path", "p", ".", "path to analyze (default: current directory)")
	repomixCmd.Flags().StringVarP(&repomixOutput, "output", "o", "output", "output directory for results")
	repomixCmd.Flags().StringVarP(&repomixRemote, "remote", "r", "", "remote repository URL or user/repo format")
	repomixCmd.Flags().StringVar(&repomixIncludePatterns, "include", "", "comma-separated glob patterns to include files")
	repomixCmd.Flags().StringVar(&repomixIgnorePatterns, "ignore", "", "comma-separated glob patterns to ignore files")

	// Output Format Options (matching the image)
	repomixCmd.Flags().BoolVar(&repomixIncludeFileSummary, "include-file-summary", true, "include file summary in output")
	repomixCmd.Flags().BoolVar(&repomixIncludeDirectoryStructure, "include-directory-structure", true, "include directory structure in output")
	repomixCmd.Flags().BoolVar(&repomixShowLineNumbers, "show-line-numbers", true, "show line numbers in output")
	repomixCmd.Flags().BoolVar(&repomixOutputParsableFormat, "output-parsable-format", true, "output in parsable format")

	// File Processing Options (matching the image)
	repomixCmd.Flags().BoolVar(&repomixRemoveComments, "remove-comments", true, "remove comments from code")
	repomixCmd.Flags().BoolVar(&repomixRemoveEmptyLines, "remove-empty-lines", true, "remove empty lines from code")

	// Additional options
	repomixCmd.Flags().IntVar(&repomixTopFilesLength, "top-files-length", 5, "number of top files to show in summary")
	repomixCmd.Flags().StringVar(&repomixHeaderText, "header-text", "", "custom header text for output")

	// Bind flags to viper
	viper.BindPFlag("repomix.path", repomixCmd.Flags().Lookup("path"))
	viper.BindPFlag("repomix.output", repomixCmd.Flags().Lookup("output"))
	viper.BindPFlag("repomix.remote", repomixCmd.Flags().Lookup("remote"))
	viper.BindPFlag("repomix.include-patterns", repomixCmd.Flags().Lookup("include"))
	viper.BindPFlag("repomix.ignore-patterns", repomixCmd.Flags().Lookup("ignore"))
	viper.BindPFlag("repomix.include-file-summary", repomixCmd.Flags().Lookup("include-file-summary"))
	viper.BindPFlag("repomix.include-directory-structure", repomixCmd.Flags().Lookup("include-directory-structure"))
	viper.BindPFlag("repomix.show-line-numbers", repomixCmd.Flags().Lookup("show-line-numbers"))
	viper.BindPFlag("repomix.output-parsable-format", repomixCmd.Flags().Lookup("output-parsable-format"))
	viper.BindPFlag("repomix.remove-comments", repomixCmd.Flags().Lookup("remove-comments"))
	viper.BindPFlag("repomix.remove-empty-lines", repomixCmd.Flags().Lookup("remove-empty-lines"))
	viper.BindPFlag("repomix.top-files-length", repomixCmd.Flags().Lookup("top-files-length"))
	viper.BindPFlag("repomix.header-text", repomixCmd.Flags().Lookup("header-text"))
}

func runRepomix(cmd *cobra.Command, args []string) error {
	// Use path from args if provided, otherwise use flag value
	targetPath := repomixPath
	if len(args) > 0 {
		targetPath = args[0]
	}

	// Check if analyzing remote repository
	if repomixRemote == "" {
		// Local repository analysis
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			return fmt.Errorf("target path does not exist: %s", targetPath)
		}
	}

	// Create output directory
	if err := createOutputDir(repomixOutput); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Check if Node.js and npm/npx are available
	if err := checkNodeAvailable(); err != nil {
		return fmt.Errorf("Node.js/npm is not available: %v", err)
	}

	// Run repomix analysis
	if repomixRemote != "" {
		fmt.Printf("Running repomix analysis on remote repository: %s\n", repomixRemote)
	} else {
		absPath, _ := filepath.Abs(targetPath)
		fmt.Printf("Running repomix analysis on: %s\n", absPath)
	}
	fmt.Printf("Output directory: %s\n", repomixOutput)

	if err := runRepomixAnalysis(targetPath); err != nil {
		return fmt.Errorf("failed to run repomix analysis: %v", err)
	}

	fmt.Println("✅ Repomix analysis completed successfully!")
	return nil
}

func checkNodeAvailable() error {
	// Check for npx first (preferred)
	if _, err := exec.LookPath("npx"); err == nil {
		if verbose {
			fmt.Println("Found npx")
		}
		return nil
	}

	// Check for npm
	if _, err := exec.LookPath("npm"); err == nil {
		if verbose {
			fmt.Println("Found npm")
		}
		return nil
	}

	// Check for node
	if _, err := exec.LookPath("node"); err == nil {
		if verbose {
			fmt.Println("Found node")
		}
		return nil
	}

	return fmt.Errorf("Node.js ecosystem not found. Please install Node.js and npm")
}

func runRepomixAnalysis(targetPath string) error {
	// Generate output filename
	timestamp := getTimestamp()
	var baseName string
	if repomixRemote != "" {
		baseName = strings.ReplaceAll(repomixRemote, "/", "_")
		baseName = strings.ReplaceAll(baseName, ":", "_")
	} else {
		baseName = filepath.Base(targetPath)
		if baseName == "." {
			baseName = "current"
		}
	}

	outputFile := filepath.Join(repomixOutput, fmt.Sprintf("%s_repomix_%s.md", baseName, timestamp))

	// Build repomix command
	args := []string{"repomix"}

	// Add target path or remote
	if repomixRemote != "" {
		args = append(args, "--remote", repomixRemote)
	} else {
		args = append(args, targetPath)
	}

	// Output settings - Force markdown format with no compression
	args = append(args, "--style", "markdown")
	args = append(args, "--output", outputFile)

	// Explicitly disable compression (we want no code compression)
	// Note: repomix has --compress flag, we don't use it to avoid compression

	// File inclusion/exclusion patterns
	if repomixIncludePatterns != "" {
		args = append(args, "--include", repomixIncludePatterns)
	}
	if repomixIgnorePatterns != "" {
		args = append(args, "--ignore", repomixIgnorePatterns)
	}

	// Output format options
	if !repomixIncludeFileSummary {
		args = append(args, "--no-file-summary")
	}
	if !repomixIncludeDirectoryStructure {
		args = append(args, "--no-directory-structure")
	}
	if repomixShowLineNumbers {
		args = append(args, "--output-show-line-numbers")
	}
	if repomixOutputParsableFormat {
		args = append(args, "--parsable-style")
	}

	// File processing options
	if repomixRemoveComments {
		args = append(args, "--remove-comments")
	}
	if repomixRemoveEmptyLines {
		args = append(args, "--remove-empty-lines")
	}

	// Additional options
	if repomixTopFilesLength != 5 {
		args = append(args, "--top-files-len", fmt.Sprintf("%d", repomixTopFilesLength))
	}
	if repomixHeaderText != "" {
		args = append(args, "--header-text", repomixHeaderText)
	}

	// Execute repomix via npx
	if verbose {
		fmt.Printf("Executing: npx %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("npx", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set working directory for local analysis
	if repomixRemote == "" {
		// Use the target path as working directory if it's a directory
		if stat, err := os.Stat(targetPath); err == nil && stat.IsDir() {
			cmd.Dir = targetPath
		} else {
			cmd.Dir = filepath.Dir(targetPath)
		}
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("repomix command failed: %v", err)
	}

	// Display results
	if info, err := os.Stat(outputFile); err == nil {
		fmt.Printf("\n📊 Analysis Results:\n")
		fmt.Printf("   Output file: %s\n", outputFile)
		fmt.Printf("   File size: %d bytes\n", info.Size())

		if repomixRemote != "" {
			fmt.Printf("   Remote repository: %s\n", repomixRemote)
		} else {
			fmt.Printf("   Local path: %s\n", targetPath)
		}

		fmt.Printf("   Format: Markdown (AI-friendly, no compression)\n")
		fmt.Printf("   Options: File Summary=%v, Directory Structure=%v, Line Numbers=%v\n",
			repomixIncludeFileSummary, repomixIncludeDirectoryStructure, repomixShowLineNumbers)
		fmt.Printf("   Processing: Remove Comments=%v, Remove Empty Lines=%v\n",
			repomixRemoveComments, repomixRemoveEmptyLines)
	}

	return nil
}
