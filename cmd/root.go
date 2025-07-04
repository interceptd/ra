package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "repo-analyzer",
	Short: "A unified tool for analyzing repositories using gitingest and repomix",
	Long: `repo-analyzer is a unified CLI tool that combines the power of gitingest and repomix
to analyze and package repositories for AI consumption.

Features:
- Run gitingest (Python) for detailed repository analysis
- Run repomix (Node.js) for AI-friendly repository packaging
- Unified configuration and output management
- Support for both local and remote repositories
- Customizable output formats and options`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is repo-analyzer.config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Bind the verbose flag to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Look for config files in current directory first
		viper.AddConfigPath(".")
		viper.SetConfigName("repo-analyzer.config")
		viper.SetConfigType("yaml")

		// Also look in home directory
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
		}
	}

	// Set verbose from viper after config is loaded
	verbose = viper.GetBool("verbose") || verbose
}

// Common utility functions are now in utils.go
func getOutputPath(baseDir, prefix, suffix string) string {
	timestamp := getTimestamp()
	filename := fmt.Sprintf("%s_%s.%s", prefix, timestamp, suffix)
	return filepath.Join(baseDir, filename)
}
