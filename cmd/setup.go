package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	setupAutoInstall bool
	setupQuiet       bool
	setupSkipPython  bool
	setupSkipNode    bool
	setupForceVenv   bool
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup and verify environment dependencies for repo-analyzer",
	Long: `Setup and verify environment dependencies for repo-analyzer.

This command performs comprehensive pre-flight checks and creates a dedicated
Python virtual environment to ensure consistent package management:

1. Python 3.x installation and accessibility
2. Virtual environment creation and activation
3. gitingest installation in isolated environment
4. Node.js and npm/npx installation
5. repomix accessibility verification

The command automatically creates a Python virtual environment in .venv/
to avoid conflicts with system Python packages.

Examples:
  # Setup with virtual environment (recommended)
  ./repo-analyzer setup
  
  # Auto-install missing dependencies
  ./repo-analyzer setup --auto-install
  
  # Force recreate virtual environment
  ./repo-analyzer setup --force-venv
  
  # Quiet mode (minimal output)
  ./repo-analyzer setup --quiet
  
  # Skip specific checks
  ./repo-analyzer setup --skip-python --skip-node`,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)

	// Setup-specific flags
	setupCmd.Flags().BoolVar(&setupAutoInstall, "auto-install", false, "automatically install missing dependencies")
	setupCmd.Flags().BoolVar(&setupQuiet, "quiet", false, "quiet mode with minimal output")
	setupCmd.Flags().BoolVar(&setupSkipPython, "skip-python", false, "skip Python and gitingest checks")
	setupCmd.Flags().BoolVar(&setupSkipNode, "skip-node", false, "skip Node.js and repomix checks")
	setupCmd.Flags().BoolVar(&setupForceVenv, "force-venv", false, "force recreate Python virtual environment")

	// Bind flags to viper
	viper.BindPFlag("setup.auto-install", setupCmd.Flags().Lookup("auto-install"))
	viper.BindPFlag("setup.quiet", setupCmd.Flags().Lookup("quiet"))
	viper.BindPFlag("setup.skip-python", setupCmd.Flags().Lookup("skip-python"))
	viper.BindPFlag("setup.skip-node", setupCmd.Flags().Lookup("skip-node"))
	viper.BindPFlag("setup.force-venv", setupCmd.Flags().Lookup("force-venv"))
}

func runSetup(cmd *cobra.Command, args []string) error {
	if !setupQuiet {
		fmt.Println("🔧 Repository Analyzer - Environment Setup")
		fmt.Println("==========================================")
		fmt.Println()
	}

	var issues []string
	var warnings []string

	// Check system information
	if !setupQuiet {
		fmt.Printf("🖥️  System Information:\n")
		fmt.Printf("   OS: %s\n", runtime.GOOS)
		fmt.Printf("   Architecture: %s\n", runtime.GOARCH)
		fmt.Printf("   Go Version: %s\n", runtime.Version())
		fmt.Println()
	}

	// Python checks
	if !setupSkipPython {
		if !setupQuiet {
			fmt.Println("🐍 Python Environment Check:")
		}

		pythonIssues, pythonWarnings := checkPythonEnvironment()
		issues = append(issues, pythonIssues...)
		warnings = append(warnings, pythonWarnings...)

		if !setupQuiet {
			fmt.Println()
		}
	}

	// Node.js checks
	if !setupSkipNode {
		if !setupQuiet {
			fmt.Println("🟢 Node.js Environment Check:")
		}

		nodeIssues, nodeWarnings := checkNodeEnvironment()
		issues = append(issues, nodeIssues...)
		warnings = append(warnings, nodeWarnings...)

		if !setupQuiet {
			fmt.Println()
		}
	}

	// Summary
	if !setupQuiet {
		fmt.Println("📊 Environment Check Summary:")
		fmt.Println("=============================")
	}

	if len(issues) == 0 {
		if !setupQuiet {
			fmt.Println("✅ All dependencies are properly installed and configured!")
			fmt.Println("🎉 Your environment is ready to use repo-analyzer!")
		}

		if len(warnings) > 0 {
			if !setupQuiet {
				fmt.Println("\n⚠️  Warnings:")
				for _, warning := range warnings {
					fmt.Printf("   • %s\n", warning)
				}
			}
		}

		return nil
	}

	// Handle issues
	if !setupQuiet {
		fmt.Printf("❌ Found %d issue(s):\n", len(issues))
		for _, issue := range issues {
			fmt.Printf("   • %s\n", issue)
		}
		fmt.Println()
	}

	// Offer to fix issues
	if setupAutoInstall {
		if !setupQuiet {
			fmt.Println("🔧 Auto-installing missing dependencies...")
		}
		return installMissingDependencies(issues)
	} else {
		if !setupQuiet {
			fmt.Println("💡 To fix these issues, you can:")
			fmt.Println("   1. Run: ./repo-analyzer setup --auto-install")
			fmt.Println("   2. Install dependencies manually (see instructions below)")
			fmt.Println("   3. Use individual commands for specific fixes")
			fmt.Println()
			printManualInstallInstructions()
		}

		if !setupQuiet {
			fmt.Print("Would you like to attempt automatic installation? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response == "y" || response == "yes" {
				return installMissingDependencies(issues)
			}
		}
	}

	return fmt.Errorf("environment setup incomplete - %d issues found", len(issues))
}

func checkPythonEnvironment() ([]string, []string) {
	var issues []string
	var warnings []string

	// Check Python installation
	pythonCmds := []string{"python3", "python"}
	var pythonCmd string

	for _, cmd := range pythonCmds {
		if _, err := exec.LookPath(cmd); err == nil {
			pythonCmd = cmd
			break
		}
	}

	if pythonCmd == "" {
		issues = append(issues, "Python is not installed or not in PATH")
		if !setupQuiet {
			fmt.Println("   ❌ Python: Not found")
		}
		return issues, warnings
	}

	// Check Python version
	cmd := exec.Command(pythonCmd, "--version")
	output, err := cmd.Output()
	if err != nil {
		issues = append(issues, "Python version check failed")
		if !setupQuiet {
			fmt.Printf("   ❌ Python: Version check failed\n")
		}
	} else {
		version := strings.TrimSpace(string(output))
		if !setupQuiet {
			fmt.Printf("   ✅ Python: %s (%s)\n", version, pythonCmd)
		}

		// Check if Python 3.x
		if !strings.Contains(version, "Python 3.") {
			warnings = append(warnings, fmt.Sprintf("Python 2.x detected (%s). Python 3.x is recommended", version))
		}
	}

	// Check virtual environment
	venvPath := getVenvPath()
	venvPython := getVenvPython()

	if setupForceVenv {
		if !setupQuiet {
			fmt.Println("   🔄 Force recreating virtual environment...")
		}
		os.RemoveAll(venvPath)
	}

	// Check if virtual environment exists
	if _, err := os.Stat(venvPython); os.IsNotExist(err) {
		issues = append(issues, "Python virtual environment not found")
		if !setupQuiet {
			fmt.Printf("   ❌ Virtual Environment: Not found (.venv/)\n")
		}
	} else {
		// Check virtual environment Python version
		cmd := exec.Command(venvPython, "--version")
		output, err := cmd.Output()
		if err != nil {
			issues = append(issues, "Virtual environment Python check failed")
			if !setupQuiet {
				fmt.Printf("   ❌ Virtual Environment: Python check failed\n")
			}
		} else {
			version := strings.TrimSpace(string(output))
			if !setupQuiet {
				fmt.Printf("   ✅ Virtual Environment: %s\n", version)
			}

			// Check gitingest in virtual environment
			cmd := exec.Command(venvPython, "-c", "import gitingest; print('gitingest available')")
			output, err := cmd.Output()
			if err != nil {
				issues = append(issues, "gitingest not installed in virtual environment")
				if !setupQuiet {
					fmt.Println("   ❌ gitingest: Not installed in virtual environment")
				}
			} else {
				if !setupQuiet {
					fmt.Printf("   ✅ gitingest: %s\n", strings.TrimSpace(string(output)))
				}
			}
		}
	}

	return issues, warnings
}

func checkNodeEnvironment() ([]string, []string) {
	var issues []string
	var warnings []string

	// Check Node.js installation
	nodeCmd := "node"
	if _, err := exec.LookPath(nodeCmd); err != nil {
		issues = append(issues, "Node.js is not installed or not in PATH")
		if !setupQuiet {
			fmt.Println("   ❌ Node.js: Not found")
		}
	} else {
		// Check Node.js version
		cmd := exec.Command(nodeCmd, "--version")
		output, err := cmd.Output()
		if err != nil {
			warnings = append(warnings, "Node.js version check failed")
			if !setupQuiet {
				fmt.Printf("   ⚠️  Node.js: Version check failed\n")
			}
		} else {
			version := strings.TrimSpace(string(output))
			if !setupQuiet {
				fmt.Printf("   ✅ Node.js: %s\n", version)
			}
		}
	}

	// Check npm installation
	npmCmd := "npm"
	if _, err := exec.LookPath(npmCmd); err != nil {
		issues = append(issues, "npm is not installed or not in PATH")
		if !setupQuiet {
			fmt.Println("   ❌ npm: Not found")
		}
	} else {
		// Check npm version
		cmd := exec.Command(npmCmd, "--version")
		output, err := cmd.Output()
		if err != nil {
			warnings = append(warnings, "npm version check failed")
			if !setupQuiet {
				fmt.Printf("   ⚠️  npm: Version check failed\n")
			}
		} else {
			version := strings.TrimSpace(string(output))
			if !setupQuiet {
				fmt.Printf("   ✅ npm: %s\n", version)
			}
		}
	}

	// Check npx installation
	npxCmd := "npx"
	if _, err := exec.LookPath(npxCmd); err != nil {
		warnings = append(warnings, "npx is not installed or not in PATH")
		if !setupQuiet {
			fmt.Println("   ⚠️  npx: Not found")
		}
	} else {
		// Check npx version
		cmd := exec.Command(npxCmd, "--version")
		output, err := cmd.Output()
		if err != nil {
			warnings = append(warnings, "npx version check failed")
			if !setupQuiet {
				fmt.Printf("   ⚠️  npx: Version check failed\n")
			}
		} else {
			version := strings.TrimSpace(string(output))
			if !setupQuiet {
				fmt.Printf("   ✅ npx: %s\n", version)
			}
		}
	}

	// Check repomix availability
	if npxCmd != "" {
		cmd := exec.Command(npxCmd, "repomix", "--version")
		output, err := cmd.Output()
		if err != nil {
			// This is expected - repomix will be installed on-demand by npx
			if !setupQuiet {
				fmt.Println("   ✅ repomix: Available via npx (will be installed on first use)")
			}
		} else {
			version := strings.TrimSpace(string(output))
			if !setupQuiet {
				fmt.Printf("   ✅ repomix: %s (already installed)\n", version)
			}
		}
	}

	return issues, warnings
}

func installMissingDependencies(issues []string) error {
	if !setupQuiet {
		fmt.Println("🔧 Installing missing dependencies...")
		fmt.Println()
	}

	for _, issue := range issues {
		if !setupQuiet {
			fmt.Printf("Fixing: %s\n", issue)
		}

		switch {
		case strings.Contains(issue, "virtual environment"):
			if err := createVirtualEnvironment(); err != nil {
				if !setupQuiet {
					fmt.Printf("❌ Failed to create virtual environment: %v\n", err)
				}
			} else {
				if !setupQuiet {
					fmt.Println("✅ Virtual environment created successfully")
				}
				// Automatically install gitingest in the new virtual environment
				if err := installGitingestInVenv(); err != nil {
					if !setupQuiet {
						fmt.Printf("❌ Failed to install gitingest in new virtual environment: %v\n", err)
					}
				} else {
					if !setupQuiet {
						fmt.Println("✅ gitingest installed successfully in new virtual environment")
					}
				}
			}

		case strings.Contains(issue, "gitingest"):
			if err := installGitingestInVenv(); err != nil {
				if !setupQuiet {
					fmt.Printf("❌ Failed to install gitingest: %v\n", err)
				}
			} else {
				if !setupQuiet {
					fmt.Println("✅ gitingest installed successfully in virtual environment")
				}
			}

		case strings.Contains(issue, "Python") && !strings.Contains(issue, "virtual"):
			if !setupQuiet {
				fmt.Println("❌ Cannot auto-install Python. Please install Python 3.x manually.")
				printPythonInstallInstructions()
			}

		case strings.Contains(issue, "Node.js"):
			if !setupQuiet {
				fmt.Println("❌ Cannot auto-install Node.js. Please install Node.js manually.")
				printNodeInstallInstructions()
			}

		case strings.Contains(issue, "npm"):
			if !setupQuiet {
				fmt.Println("❌ Cannot auto-install npm. Please install Node.js (includes npm) manually.")
				printNodeInstallInstructions()
			}
		}

		if !setupQuiet {
			fmt.Println()
		}
	}

	// Re-check environment after installation attempts
	if !setupQuiet {
		fmt.Println("🔍 Re-checking environment after installation...")
		fmt.Println()
	}

	// Verify virtual environment and gitingest
	venvPython := getVenvPython()
	if _, err := os.Stat(venvPython); err == nil {
		cmd := exec.Command(venvPython, "-c", "import gitingest; print('gitingest available')")
		output, err := cmd.Output()
		if err == nil {
			if !setupQuiet {
				fmt.Printf("✅ gitingest verification: %s\n", strings.TrimSpace(string(output)))
				fmt.Println("🎉 Environment setup completed successfully!")
			}
			return nil
		} else {
			if !setupQuiet {
				fmt.Println("❌ gitingest installation verification failed")
			}
		}
	}

	return fmt.Errorf("installation completed but some packages may still need manual setup")
}

func createVirtualEnvironment() error {
	// Find Python executable
	pythonCmds := []string{"python3", "python"}
	var pythonCmd string

	for _, cmd := range pythonCmds {
		if _, err := exec.LookPath(cmd); err == nil {
			pythonCmd = cmd
			break
		}
	}

	if pythonCmd == "" {
		return fmt.Errorf("no Python executable found")
	}

	venvPath := getVenvPath()

	// Remove existing virtual environment if it exists
	if setupForceVenv {
		os.RemoveAll(venvPath)
	}

	// Create virtual environment
	cmd := exec.Command(pythonCmd, "-m", "venv", venvPath)
	if !setupQuiet {
		fmt.Printf("   Running: %s -m venv %s\n", pythonCmd, venvPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if !setupQuiet {
			fmt.Printf("   Output: %s\n", string(output))
		}
		return fmt.Errorf("failed to create virtual environment: %v", err)
	}

	if !setupQuiet && verbose {
		fmt.Printf("   Output: %s\n", string(output))
	}

	return nil
}

func installGitingestInVenv() error {
	venvPip := getVenvPip()

	// Check if virtual environment exists
	if _, err := os.Stat(venvPip); os.IsNotExist(err) {
		// Create virtual environment first
		if err := createVirtualEnvironment(); err != nil {
			return fmt.Errorf("failed to create virtual environment: %v", err)
		}
	}

	// Upgrade pip first to avoid potential issues
	if !setupQuiet {
		fmt.Printf("   Upgrading pip in virtual environment...\n")
	}

	upgradePipCmd := exec.Command(venvPip, "install", "--upgrade", "pip")
	upgradePipOutput, err := upgradePipCmd.CombinedOutput()
	if err != nil {
		if !setupQuiet {
			fmt.Printf("   ⚠️  Warning: Failed to upgrade pip: %v\n", err)
			fmt.Printf("   Pip upgrade output: %s\n", string(upgradePipOutput))
		}
		// Continue anyway - this is not a critical error
	}

	// Install gitingest in virtual environment
	cmd := exec.Command(venvPip, "install", "gitingest")
	if !setupQuiet {
		fmt.Printf("   Running: %s install gitingest\n", venvPip)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if !setupQuiet {
			fmt.Printf("   ❌ Installation failed with error: %v\n", err)
			fmt.Printf("   Full output: %s\n", string(output))
		}
		return fmt.Errorf("failed to install gitingest: %v", err)
	}

	if !setupQuiet {
		fmt.Printf("   ✅ Installation output: %s\n", string(output))
	}

	return nil
}

func printManualInstallInstructions() {
	fmt.Println("📋 Manual Installation Instructions:")
	fmt.Println("===================================")

	printPythonInstallInstructions()
	printVenvInstallInstructions()
	printGitingestInstallInstructions()
	printNodeInstallInstructions()
}

func printPythonInstallInstructions() {
	fmt.Println("🐍 Python Installation:")
	switch runtime.GOOS {
	case "darwin":
		fmt.Println("   # macOS")
		fmt.Println("   brew install python3")
		fmt.Println("   # or download from https://www.python.org/downloads/")
	case "linux":
		fmt.Println("   # Ubuntu/Debian")
		fmt.Println("   sudo apt update && sudo apt install python3 python3-pip python3-venv")
		fmt.Println("   # CentOS/RHEL")
		fmt.Println("   sudo yum install python3 python3-pip python3-venv")
	case "windows":
		fmt.Println("   # Windows")
		fmt.Println("   # Download from https://www.python.org/downloads/")
		fmt.Println("   # or use winget: winget install Python.Python.3")
	}
	fmt.Println()
}

func printVenvInstallInstructions() {
	fmt.Println("📦 Virtual Environment Setup:")
	fmt.Println("   # Create virtual environment")
	fmt.Println("   python3 -m venv .venv")
	fmt.Println("   # Activate virtual environment")
	switch runtime.GOOS {
	case "windows":
		fmt.Println("   .venv\\Scripts\\activate")
	default:
		fmt.Println("   source .venv/bin/activate")
	}
	fmt.Println()
}

func printGitingestInstallInstructions() {
	fmt.Println("🔍 gitingest Installation:")
	fmt.Println("   # In virtual environment")
	fmt.Println("   .venv/bin/pip install gitingest")
	fmt.Println("   # or manually")
	fmt.Println("   pip3 install gitingest")
	fmt.Println()
}

func printNodeInstallInstructions() {
	fmt.Println("🟢 Node.js Installation:")
	switch runtime.GOOS {
	case "darwin":
		fmt.Println("   # macOS")
		fmt.Println("   brew install node")
		fmt.Println("   # or download from https://nodejs.org/")
	case "linux":
		fmt.Println("   # Ubuntu/Debian")
		fmt.Println("   curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -")
		fmt.Println("   sudo apt-get install -y nodejs")
		fmt.Println("   # or")
		fmt.Println("   sudo snap install node --classic")
	case "windows":
		fmt.Println("   # Windows")
		fmt.Println("   # Download from https://nodejs.org/")
		fmt.Println("   # or use winget: winget install OpenJS.NodeJS")
	}
	fmt.Println()
}
