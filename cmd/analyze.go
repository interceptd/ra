package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Analyze command flags
var (
	analyzePath       string
	analyzeOutput     string
	analyzeGitingest  bool
	analyzeRepomix    bool
	analyzeHeaderText string
)

// analyzeCmd represents the analyze command
var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Run comprehensive repository analysis using both gitingest and repomix",
	Long: `Run comprehensive repository analysis using both gitingest and repomix tools.
This command combines the power of both tools to provide:
- Detailed repository documentation (gitingest)
- AI-friendly markdown package (repomix)

Examples:
  # Analyze current directory
  repo-analyzer analyze

  # Analyze specific directory
  repo-analyzer analyze ./my-project

  # Analyze with custom output directory
  repo-analyzer analyze ./my-project --output ./analysis-results

  # Run only gitingest analysis
  repo-analyzer analyze --enable-gitingest --disable-repomix

  # Run only repomix analysis
  repo-analyzer analyze --enable-repomix --disable-gitingest

  # Analyze remote repository
  repo-analyzer analyze https://github.com/user/repo`,
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	// Analyze-specific flags
	analyzeCmd.Flags().StringVarP(&analyzePath, "path", "p", ".", "path to analyze (default: current directory)")
	analyzeCmd.Flags().StringVarP(&analyzeOutput, "output", "o", "analysis-results", "output directory for results")
	analyzeCmd.Flags().BoolVar(&analyzeGitingest, "enable-gitingest", true, "enable gitingest analysis")
	analyzeCmd.Flags().BoolVar(&analyzeRepomix, "enable-repomix", true, "enable repomix analysis")
	analyzeCmd.Flags().StringVar(&analyzeHeaderText, "header-text", "", "custom header text for output")

	// Convenience flags to disable features
	analyzeCmd.Flags().BoolVar(&analyzeGitingest, "disable-gitingest", false, "disable gitingest analysis")
	analyzeCmd.Flags().BoolVar(&analyzeRepomix, "disable-repomix", false, "disable repomix analysis")

	// Mark disable flags as mutually exclusive with enable flags
	analyzeCmd.MarkFlagsMutuallyExclusive("enable-gitingest", "disable-gitingest")
	analyzeCmd.MarkFlagsMutuallyExclusive("enable-repomix", "disable-repomix")

	// Bind flags to viper
	viper.BindPFlag("analyze.path", analyzeCmd.Flags().Lookup("path"))
	viper.BindPFlag("analyze.output", analyzeCmd.Flags().Lookup("output"))
	viper.BindPFlag("analyze.enable_gitingest", analyzeCmd.Flags().Lookup("enable-gitingest"))
	viper.BindPFlag("analyze.enable_repomix", analyzeCmd.Flags().Lookup("enable-repomix"))
	viper.BindPFlag("analyze.header_text", analyzeCmd.Flags().Lookup("header-text"))
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// Use path from args if provided, otherwise use flag value
	targetPath := analyzePath
	if len(args) > 0 {
		targetPath = args[0]
	}

	// Use config file values if flags are not set
	if !cmd.Flags().Changed("output") {
		if configOutput := viper.GetString("analyze.output_dir"); configOutput != "" {
			analyzeOutput = configOutput
		}
	}

	// Handle enable/disable flags
	if cmd.Flags().Changed("disable-gitingest") {
		analyzeGitingest = false
	} else if !cmd.Flags().Changed("enable-gitingest") {
		// Use config value if flag not changed, otherwise keep default
		if viper.IsSet("analyze.enable_gitingest") {
			analyzeGitingest = viper.GetBool("analyze.enable_gitingest")
		}
	}

	if cmd.Flags().Changed("disable-repomix") {
		analyzeRepomix = false
	} else if !cmd.Flags().Changed("enable-repomix") {
		// Use config value if flag not changed, otherwise keep default
		if viper.IsSet("analyze.enable_repomix") {
			analyzeRepomix = viper.GetBool("analyze.enable_repomix")
		}
	}

	if !cmd.Flags().Changed("header-text") {
		if configHeader := viper.GetString("analyze.header_text"); configHeader != "" {
			analyzeHeaderText = configHeader
		}
	}

	// Check if it's a remote URL
	isRemote := isRemoteRepository(targetPath)

	var absPath string
	var err error

	if isRemote {
		// Handle remote repository
		fmt.Printf("🌐 Remote repository detected: %s\n", targetPath)
		absPath = targetPath
	} else {
		// Handle local path
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			return fmt.Errorf("target path does not exist: %s", targetPath)
		}

		absPath, err = filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %v", err)
		}
	}

	// Create output directory
	if err := createOutputDir(analyzeOutput); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Print analysis configuration
	fmt.Printf("🔍 Starting comprehensive repository analysis...\n")
	if isRemote {
		fmt.Printf("Target: Remote repository - %s\n", absPath)
	} else {
		fmt.Printf("Target: Local path - %s\n", absPath)
	}
	fmt.Printf("Output directory: %s\n", analyzeOutput)
	fmt.Printf("Tools to run: gitingest=%t, repomix=%t\n", analyzeGitingest, analyzeRepomix)

	// Validate that at least one tool is enabled
	if !analyzeGitingest && !analyzeRepomix {
		return fmt.Errorf("at least one analysis tool must be enabled")
	}

	// Track results
	var results []string
	hasErrors := false

	// Run gitingest analysis
	if analyzeGitingest {
		fmt.Printf("\n📝 Running gitingest analysis...\n")
		if err := runGitingestForAnalyze(absPath, isRemote); err != nil {
			fmt.Printf("❌ Gitingest analysis failed: %v\n", err)
			hasErrors = true
		} else {
			fmt.Printf("✅ Gitingest analysis completed successfully!\n")
			results = append(results, "✅ Gitingest: SUCCESS - Detailed repository documentation generated")
		}
	}

	// Run repomix analysis
	if analyzeRepomix {
		fmt.Printf("\n📦 Running repomix analysis...\n")
		if err := runRepomixForAnalyze(absPath, isRemote); err != nil {
			fmt.Printf("❌ Repomix analysis failed: %v\n", err)
			hasErrors = true
		} else {
			fmt.Printf("✅ Repomix analysis completed successfully!\n")
			results = append(results, "✅ Repomix: SUCCESS - AI-friendly markdown package generated")
		}
	}

	// Print summary
	fmt.Printf("\n📊 Analysis Summary:\n")
	for _, result := range results {
		fmt.Printf("   %s\n", result)
	}

	if hasErrors {
		fmt.Printf("\n⚠️  Some analyses failed. Check the output above for details.\n")
		return fmt.Errorf("analysis completed with errors")
	}

	fmt.Printf("\n🎉 Repository analysis completed! Results saved to: %s\n", analyzeOutput)
	return nil
}

func runGitingestForAnalyze(targetPath string, isRemote bool) error {
	// Set gitingest parameters
	gitingestPath = targetPath
	gitingestOutput = analyzeOutput
	gitingestJSONOnly = false

	// If remote, we need to handle it differently
	if isRemote {
		// For remote repositories, gitingest might need special handling
		// For now, we'll try to pass the URL directly
		fmt.Printf("Attempting to analyze remote repository with gitingest...\n")
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
	return runGitingestAnalysis(targetPath)
}

func runRepomixForAnalyze(targetPath string, isRemote bool) error {
	// Set repomix parameters
	repomixPath = targetPath
	repomixOutput = analyzeOutput
	repomixIncludeFileSummary = viper.GetBool("repomix.include_file_summary")
	repomixIncludeDirectoryStructure = viper.GetBool("repomix.include_directory_structure")
	repomixShowLineNumbers = viper.GetBool("repomix.show_line_numbers")
	repomixOutputParsableFormat = viper.GetBool("repomix.output_parsable_format")
	repomixRemoveComments = viper.GetBool("repomix.remove_comments")
	repomixRemoveEmptyLines = viper.GetBool("repomix.remove_empty_lines")
	repomixTopFilesLength = viper.GetInt("repomix.top_files_length")

	// Set remote URL if analyzing remote repository
	if isRemote {
		repomixRemote = targetPath
	} else {
		repomixRemote = ""
	}

	// Use analyze header text if set, otherwise use repomix config
	if analyzeHeaderText != "" {
		repomixHeaderText = analyzeHeaderText
	} else {
		repomixHeaderText = viper.GetString("repomix.header_text")
	}

	// Check if Node.js is available
	if err := checkNodeAvailable(); err != nil {
		return fmt.Errorf("Node.js/npm is not available: %v", err)
	}

	// Run repomix analysis
	return runRepomixAnalysis(targetPath)
}

// isRemoteRepository determines if a path is a remote repository URL or shorthand
func isRemoteRepository(path string) bool {
	// Check for full URLs first
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		return true
	}

	// Exclude local paths (most common cases)
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "../") {
		return false
	}

	// Check for GitHub shorthand format (user/repo)
	// Must contain exactly one "/" and be a simple name without dots
	if strings.Contains(path, "/") && strings.Count(path, "/") == 1 && !strings.Contains(path, ".") {
		return true
	}

	return false
}
