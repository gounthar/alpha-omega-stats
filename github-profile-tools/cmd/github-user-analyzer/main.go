package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenkins/github-profile-tools/internal/markdown"
	"github.com/jenkins/github-profile-tools/internal/profile"
	"github.com/joho/godotenv"
)

const (
	version = "1.0.0"
)

// Config holds command line configuration
type Config struct {
	Username     string
	Token        string
	OutputDir    string
	Template     string
	Format       string
	Verbose      bool
	SaveJSON     bool
	ShowVersion  bool
	Timeout      time.Duration
	DebugLogFile string
}

// main is the entry point for the GitHub User Analyzer CLI.
// It loads environment variables from ../.env or .env, parses and validates command-line flags,
// optionally prints the tool version and exits, configures dual debug logging, creates a context
// with the configured timeout, and runs the profile analysis, terminating the program on fatal errors.
func main() {
	// Load .env file if it exists
	if err := godotenv.Load("../.env"); err != nil {
		// Try loading from current directory
		if err := godotenv.Load(".env"); err != nil {
			// .env file not found, continue without it
		}
	}

	config := parseFlags()

	if config.ShowVersion {
		fmt.Printf("GitHub User Analyzer v%s\n", version)
		os.Exit(0)
	}

	if err := validateConfig(config); err != nil {
		log.Fatal(err)
	}

	// Set up dual logging (console + file)
	logFile, err := setupDebugLogging(config.DebugLogFile, config.Verbose)
	if err != nil {
		log.Printf("Warning: Failed to set up debug logging: %v", err)
	} else {
		defer logFile.Close()
		log.Printf("Debug logging enabled: %s", config.DebugLogFile)
	}

	if config.Verbose {
		log.Printf("Using timeout: %v", config.Timeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	if err := runAnalysis(ctx, config); err != nil {
		log.Fatal(err)
	}
}

// parseFlags parses command-line flags and returns a Config populated from flag values and environment variables.
// The function registers flags for username, token, output directory, template, format, verbosity, JSON saving, version, timeout, and debug log file;
// it provides a custom usage message, resolves the timeout via parseTimeout, and selects DebugLogFile from the flag, the DEBUG_LOG_FILE environment variable, or a sensible default before returning the populated Config.
func parseFlags() Config {
	config := Config{}

	var timeoutStr string

	flag.StringVar(&config.Username, "user", "", "GitHub username to analyze (required)")
	flag.StringVar(&config.Token, "token", os.Getenv("GITHUB_TOKEN"), "GitHub API token (or set GITHUB_TOKEN env var)")
	flag.StringVar(&config.OutputDir, "output", "./data/profiles", "Output directory for generated files")
	flag.StringVar(&config.Template, "template", "all", "Template type: resume, technical, executive, ats, all (default: all)")
	flag.StringVar(&config.Format, "format", "both", "Output format: markdown, json, both")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&config.SaveJSON, "save-json", true, "Save raw JSON profile data")
	flag.BoolVar(&config.ShowVersion, "version", false, "Show version and exit")
	flag.StringVar(&timeoutStr, "timeout", "", "Analysis timeout (e.g., '30m', '2h', '6h'). Default: 6h, or set ANALYSIS_TIMEOUT env var")
	flag.StringVar(&config.DebugLogFile, "debug-log", "", "Debug log file path (default: github-user-analyzer-debug.log, or set DEBUG_LOG_FILE env var)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "GitHub User Analyzer v%s\n\n", version)
		fmt.Fprintf(os.Stderr, "Generate professional profiles from GitHub user data.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s -user USERNAME [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s -user octocat                           # Generate all profile templates\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -user octocat -template resume          # Generate only resume template\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -user octocat -template technical       # Generate only technical template\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -user octocat -format markdown          # Generate all templates in markdown only\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -user octocat -output ./resumes         # Generate all templates in custom directory\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -user octocat -timeout 2h -verbose      # Generate all templates with extended timeout\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Parse timeout from command line, environment variable, or use default
	config.Timeout = parseTimeout(timeoutStr)

	// Set debug log file from command line flag, environment variable, or default
	if config.DebugLogFile == "" {
		config.DebugLogFile = os.Getenv("DEBUG_LOG_FILE")
	}
	if config.DebugLogFile == "" {
		config.DebugLogFile = "github-user-analyzer-debug.log"
	}


	return config
}

// setupDebugLogging sets up dual logging to both console and file
func setupDebugLogging(debugLogFile string, verbose bool) (*os.File, error) {
	// Create or open the debug log file
	logFile, err := os.OpenFile(debugLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open debug log file %s: %w", debugLogFile, err)
	}

	// Write a session separator to the log file
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	sessionHeader := fmt.Sprintf("\n=== GitHub User Analyzer Debug Session - %s ===\n", timestamp)
	if _, err := logFile.WriteString(sessionHeader); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to write session header: %w", err)
	}

	// Set up dual output: console + file
	var writers []io.Writer
	writers = append(writers, os.Stderr) // Console output
	writers = append(writers, logFile)   // File output

	multiWriter := io.MultiWriter(writers...)
	log.SetOutput(multiWriter)

	// Set log format with timestamps and file info if verbose
	if verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	return logFile, nil
}

// parseTimeout parses timeout from command line flag, environment variable, or returns default
func parseTimeout(flagValue string) time.Duration {
	// Default timeout is 6 hours
	defaultTimeout := 6 * time.Hour

	// Priority: command line flag -> environment variable -> default
	timeoutStr := flagValue
	if timeoutStr == "" {
		timeoutStr = os.Getenv("ANALYSIS_TIMEOUT")
	}
	if timeoutStr == "" {
		return defaultTimeout
	}

	// Parse the duration string
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		log.Printf("Warning: Invalid timeout format '%s', using default %v", timeoutStr, defaultTimeout)
		return defaultTimeout
	}

	// Validate reasonable bounds (1 minute to 24 hours)
	if timeout < time.Minute {
		log.Printf("Warning: Timeout too short (%v), using 1 minute minimum", timeout)
		return time.Minute
	}
	if timeout > 24*time.Hour {
		log.Printf("Warning: Timeout too long (%v), using 24 hour maximum", timeout)
		return 24 * time.Hour
	}

	return timeout
}

// validateConfig validates the configuration
func validateConfig(config Config) error {
	if config.Username == "" {
		return fmt.Errorf("username is required (use -user flag)")
	}

	if config.Token == "" {
		return fmt.Errorf("GitHub token is required (use -token flag or set GITHUB_TOKEN environment variable)")
	}

	validTemplates := []string{"resume", "technical", "executive", "ats", "all"}
	if !contains(validTemplates, config.Template) {
		return fmt.Errorf("invalid template: %s (valid options: %s)", config.Template, strings.Join(validTemplates, ", "))
	}

	validFormats := []string{"markdown", "json", "both"}
	if !contains(validFormats, config.Format) {
		return fmt.Errorf("invalid format: %s (valid options: %s)", config.Format, strings.Join(validFormats, ", "))
	}

	return nil
}

// runAnalysis performs the GitHub user analysis
func runAnalysis(ctx context.Context, config Config) error {
	if config.Verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Printf("Starting analysis for user: %s", config.Username)
		log.Printf("Using template: %s", config.Template)
		log.Printf("Output directory: %s", config.OutputDir)
	}

	// Create analyzer
	analyzer := profile.NewAnalyzer(config.Token)

	// Analyze user profile
	log.Printf("Analyzing GitHub profile for user: %s", config.Username)
	prof, err := analyzer.AnalyzeUser(ctx, config.Username)
	if err != nil {
		return fmt.Errorf("failed to analyze user %s: %w", config.Username, err)
	}

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate outputs based on format
	if config.Format == "json" || config.Format == "both" {
		if err := saveJSONProfile(prof, config); err != nil {
			return fmt.Errorf("failed to save JSON profile: %w", err)
		}
	}

	if config.Format == "markdown" || config.Format == "both" {
		// Determine which templates to generate
		var templatesToGenerate []string
		if config.Template == "all" {
			templatesToGenerate = []string{"resume", "technical", "executive", "ats"}
		} else {
			templatesToGenerate = []string{config.Template}
		}

		// Generate each template
		for _, template := range templatesToGenerate {
			templateConfig := config
			templateConfig.Template = template
			if err := generateMarkdownProfile(prof, templateConfig); err != nil {
				return fmt.Errorf("failed to generate %s markdown profile: %w", template, err)
			}
		}
	}

	// Print summary
	printSummary(prof, config)

	// Print final rate limit status
	if config.Verbose {
		rateLimitStatus := analyzer.GetGitHubRateLimitStatus()
		log.Printf("Final GitHub API Rate Limit Status:")
		log.Printf("  Resource: %s", rateLimitStatus.Resource)
		log.Printf("  Used: %d/%d requests", rateLimitStatus.Used, rateLimitStatus.Limit)
		log.Printf("  Remaining: %d requests", rateLimitStatus.Remaining)
		log.Printf("  Resets at: %s", rateLimitStatus.ResetTime.Format("15:04:05 MST"))

		percentUsed := float64(rateLimitStatus.Used) / float64(rateLimitStatus.Limit) * 100
		log.Printf("  Usage: %.1f%% of hourly quota", percentUsed)
	}

	return nil
}

// saveJSONProfile saves the profile data as JSON
func saveJSONProfile(prof *profile.UserProfile, config Config) error {
	filename := fmt.Sprintf("%s_profile.json", prof.Username)
	filepath := filepath.Join(config.OutputDir, filename)

	data, err := json.MarshalIndent(prof, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile to JSON: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	if config.Verbose {
		log.Printf("Saved JSON profile: %s", filepath)
	}

	return nil
}

// generateMarkdownProfile generates and saves the markdown profile
func generateMarkdownProfile(prof *profile.UserProfile, config Config) error {
	generator := markdown.NewGenerator()

	templateType := markdown.TemplateType(config.Template)
	content, err := generator.GenerateMarkdown(prof, templateType)
	if err != nil {
		return fmt.Errorf("failed to generate markdown: %w", err)
	}

	filename := fmt.Sprintf("%s_profile_%s.md", prof.Username, config.Template)
	filepath := filepath.Join(config.OutputDir, filename)

	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}

	if config.Verbose {
		log.Printf("Generated markdown profile: %s", filepath)
	}

	return nil
}

// printSummary prints a summary of the analysis
func printSummary(prof *profile.UserProfile, config Config) {
	fmt.Printf("\n🎉 Analysis Complete for @%s\n", prof.Username)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if prof.Name != "" {
		fmt.Printf("👤 Name: %s\n", prof.Name)
	}

	if prof.Company != "" {
		fmt.Printf("🏢 Company: %s\n", prof.Company)
	}

	if prof.Location != "" {
		fmt.Printf("📍 Location: %s\n", prof.Location)
	}

	fmt.Printf("\n📊 Profile Statistics:\n")
	fmt.Printf("   • Career Level: %s\n", strings.Title(prof.Insights.CareerLevel))
	fmt.Printf("   • Experience: %d years\n", prof.Contributions.ContributionYears)
	fmt.Printf("   • Repositories: %d total\n", len(prof.Repositories))

	totalStars := 0
	ownedRepos := 0
	for _, repo := range prof.Repositories {
		totalStars += repo.Stars
		if repo.IsOwner {
			ownedRepos++
		}
	}

	fmt.Printf("   • Community Impact: %d stars received\n", totalStars)
	fmt.Printf("   • Repository Ownership: %d owned projects\n", ownedRepos)
	fmt.Printf("   • Organizations: %d active memberships\n", len(prof.Organizations))

	if len(prof.Skills.PrimaryLanguages) > 0 {
		fmt.Printf("\n🛠  Primary Technologies:\n")
		for i, lang := range prof.Skills.PrimaryLanguages {
			if i >= 5 { // Show top 5
				break
			}
			// Find the language stats
			for _, langStats := range prof.Languages {
				if strings.EqualFold(langStats.Language, lang) {
					fmt.Printf("   • %s (%.1f%% of codebase)\n", lang, langStats.Percentage)
					break
				}
			}
		}
	}

	if len(prof.Insights.RecommendedRoles) > 0 {
		fmt.Printf("\n💼 Recommended Roles:\n")
		for i, role := range prof.Insights.RecommendedRoles {
			if i >= 3 { // Show top 3
				break
			}
			fmt.Printf("   • %s\n", role)
		}
	}

	fmt.Printf("\n📁 Output Files:\n")

	if config.Format == "json" || config.Format == "both" {
		jsonFile := fmt.Sprintf("%s_profile.json", prof.Username)
		fmt.Printf("   • JSON Data: %s\n", filepath.Join(config.OutputDir, jsonFile))
	}

	if config.Format == "markdown" || config.Format == "both" {
		if config.Template == "all" {
			templates := []string{"resume", "technical", "executive", "ats"}
			for _, template := range templates {
				mdFile := fmt.Sprintf("%s_profile_%s.md", prof.Username, template)
				fmt.Printf("   • Markdown Profile (%s): %s\n", template, filepath.Join(config.OutputDir, mdFile))
			}
		} else {
			mdFile := fmt.Sprintf("%s_profile_%s.md", prof.Username, config.Template)
			fmt.Printf("   • Markdown Profile: %s\n", filepath.Join(config.OutputDir, mdFile))
		}
	}

	fmt.Printf("\n✨ Impact Score: %.1f/10\n", prof.Insights.OverallImpactScore*10)

	fmt.Printf("\nℹ️  Template Options:\n")
	if config.Template == "all" {
		fmt.Printf("   • Generated all templates by default (resume, technical, executive, ats)\n")
		fmt.Printf("   • Use --template [type] to generate a specific template only\n")
	} else {
		fmt.Printf("   • --template resume     (General resume enhancement)\n")
		fmt.Printf("   • --template technical  (Deep technical analysis)\n")
		fmt.Printf("   • --template executive  (Leadership focus)\n")
		fmt.Printf("   • --template ats        (ATS/Applicant Tracking System optimized)\n")
		fmt.Printf("   • --template all        (Generate all templates - default behavior)\n")
	}

	fmt.Printf("\n🚀 Ready to enhance your resume with GitHub data!\n")
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}