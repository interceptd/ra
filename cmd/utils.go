package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"
)

// createOutputDir creates the output directory if it doesn't exist
func createOutputDir(outputDir string) error {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}
		if verbose {
			fmt.Printf("Created output directory: %s\n", outputDir)
		}
	}
	return nil
}

// getTimestamp returns a timestamp string in the format YYYYMMDD_HHMMSS
func getTimestamp() string {
	return time.Now().Format("20060102_150405")
}

// formatFileSize formats file size in human-readable format
func formatFileSize(size int64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// printSeparator prints a visual separator line
func printSeparator() {
	if verbose {
		fmt.Println("────────────────────────────────────────────────────────────────")
	}
}

// printHeader prints a formatted header
func printHeader(title string) {
	if verbose {
		fmt.Printf("\n%s\n", title)
		printSeparator()
	}
}

// cleanTextContent removes unusual Unicode line terminators and normalizes text
func cleanTextContent(content string) string {
	// Remove Line Separator (U+2028) and Paragraph Separator (U+2029)
	content = strings.ReplaceAll(content, "\u2028", "\n") // LS -> LF
	content = strings.ReplaceAll(content, "\u2029", "\n") // PS -> LF

	// Remove other problematic Unicode characters
	content = strings.ReplaceAll(content, "\u200B", "") // Zero Width Space
	content = strings.ReplaceAll(content, "\u200C", "") // Zero Width Non-Joiner
	content = strings.ReplaceAll(content, "\u200D", "") // Zero Width Joiner
	content = strings.ReplaceAll(content, "\uFEFF", "") // Zero Width No-Break Space (BOM)

	// Normalize line endings to Unix style (LF only)
	content = strings.ReplaceAll(content, "\r\n", "\n") // CRLF -> LF
	content = strings.ReplaceAll(content, "\r", "\n")   // CR -> LF

	return content
}

// cleanTextForFile removes unusual characters and normalizes content for file writing
func cleanTextForFile(content string) string {
	// Basic cleaning
	cleaned := cleanTextContent(content)

	// Remove any remaining control characters except newlines, tabs, and carriage returns
	var builder strings.Builder
	builder.Grow(len(cleaned))

	for _, r := range cleaned {
		// Keep printable characters, newlines, tabs, and spaces
		if unicode.IsPrint(r) || r == '\n' || r == '\t' || r == ' ' {
			builder.WriteRune(r)
		} else if unicode.IsSpace(r) {
			// Convert other whitespace to regular space
			builder.WriteRune(' ')
		}
		// Skip other control characters
	}

	return builder.String()
}

// writeCleanFile writes content to a file with line terminator cleaning
func writeCleanFile(filename, content string) error {
	cleanedContent := cleanTextForFile(content)

	err := os.WriteFile(filename, []byte(cleanedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %v", filename, err)
	}

	if verbose {
		fmt.Printf("✅ File written with cleaned content: %s\n", filename)
	}

	return nil
}

// writeCleanJSONFile writes JSON content to a file with cleaning
func writeCleanJSONFile(filename, content string) error {
	// For JSON, we're more conservative - just clean line separators
	cleanedContent := strings.ReplaceAll(content, "\u2028", "\\n")
	cleanedContent = strings.ReplaceAll(cleanedContent, "\u2029", "\\n")

	err := os.WriteFile(filename, []byte(cleanedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file %s: %v", filename, err)
	}

	if verbose {
		fmt.Printf("✅ JSON file written with cleaned content: %s\n", filename)
	}

	return nil
}

// getVenvPath returns the path to the Python virtual environment
func getVenvPath() string {
	// Check if we're running in a container (look for pre-built venv)
	if _, err := os.Stat("/app/.venv"); err == nil {
		return "/app/.venv"
	}

	// Fallback to local .venv in current directory
	cwd, err := os.Getwd()
	if err != nil {
		return ".venv"
	}
	return filepath.Join(cwd, ".venv")
}

// getVenvPython returns the path to the Python executable in the virtual environment
func getVenvPython() string {
	venvPath := getVenvPath()
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts", "python.exe")
	}
	return filepath.Join(venvPath, "bin", "python")
}

// getVenvPip returns the path to the pip executable in the virtual environment
func getVenvPip() string {
	venvPath := getVenvPath()
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts", "pip.exe")
	}
	return filepath.Join(venvPath, "bin", "pip")
}
