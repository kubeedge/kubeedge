/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package main implements the KubeEdge Auto Test Generator
// Following mentor's workflow: PR Merged → Check Coverage → Generate Tests → Validate → Create PR
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Config holds the simplified configuration
type Config struct {
	CoverageThreshold float64
	MaxRetries        int
	GeminiAPIKey      string
	ChangedFiles      string
	WorkingDir        string
	Debug             bool
	// PR Creation settings
	CreatePR         bool
	GitHubToken      string
	RepoOwner        string
	RepoName         string
}

// ProcessResult represents the result of processing a single file
type ProcessResult struct {
	SourceFile     string
	TestFile       string
	Success        bool
	Error          error
	BeforeCoverage float64
	AfterCoverage  float64
	Duration       time.Duration
}

func main() {
	// Parse command line flags
	config := parseFlags()

	if config.Debug {
		log.Printf("🚀 Starting KubeEdge Auto Test Generator")
		log.Printf("📋 Coverage Threshold: %.0f%%", config.CoverageThreshold)
	}

	ctx := context.Background()

	// Initialize components with proper constructors
	validator := NewTestValidator(config.WorkingDir, config.CoverageThreshold)
	generator := NewKubeEdgeTestGenerator(config.GeminiAPIKey)
	defer generator.Close()

	// Step 1: Get modified files (from git or provided list)
	var filesToCheck []string
	var err error

	if config.ChangedFiles != "" {
		// Use provided files (from workflow)
		filesToCheck = parseChangedFiles(config.ChangedFiles)
		log.Printf("📂 Using provided files: %v", filesToCheck)
	} else {
		// Get from git (for local testing)
		filesToCheck, err = validator.GetModifiedFilesFromGit(ctx)
		if err != nil {
			log.Fatalf("❌ Failed to get modified files: %v", err)
		}
		log.Printf("📂 Found %d modified Go files", len(filesToCheck))
	}

	if len(filesToCheck) == 0 {
		log.Println("ℹ️ No Go files to process")
		return
	}

	// Step 2: Check Coverage of Modified Files → If < 40%
	lowCoverageFiles, coverageInfos, err := validator.FilterLowCoverageFiles(ctx, filesToCheck)
	if err != nil {
		log.Fatalf("❌ Failed to filter low coverage files: %v", err)
	}

	if len(lowCoverageFiles) == 0 {
		log.Println("✅ All files have sufficient coverage")
		printCoverageSummary(coverageInfos, config.CoverageThreshold)
		return
	}

	log.Printf("🎯 Found %d files needing test generation", len(lowCoverageFiles))

	// Step 3: Generate Tests → Add filename_test.go → Run go test → If Tests Pass
	var results []ProcessResult
	successCount := 0
	failureCount := 0

	for _, sourceFile := range lowCoverageFiles {
		log.Printf("\n🔄 Processing: %s", sourceFile)

		result := processFileComplete(ctx, sourceFile, config, generator, validator)
		results = append(results, result)

		if result.Success {
			successCount++
			improvement := result.AfterCoverage - result.BeforeCoverage
			log.Printf("✅ SUCCESS: %s (%.2f%% → %.2f%%, +%.2f%%)",
				sourceFile, result.BeforeCoverage, result.AfterCoverage, improvement)
		} else {
			failureCount++
			log.Printf("❌ FAILED: %s - %v", sourceFile, result.Error)
		}
	}

	// Step 4: Generate summary files for workflow
	generateWorkflowOutput(results)

	// Step 5: Create PR if requested and tests were successful
	if config.CreatePR && successCount > 0 {
		if config.GitHubToken == "" || config.RepoOwner == "" || config.RepoName == "" {
			log.Printf("⚠️ PR creation requested but missing required parameters:")
			log.Printf("   - GitHub Token: %s", boolToStatus(config.GitHubToken != ""))
			log.Printf("   - Repo Owner: %s", boolToStatus(config.RepoOwner != ""))
			log.Printf("   - Repo Name: %s", boolToStatus(config.RepoName != ""))
			log.Printf("   Skipping PR creation...")
		} else {
			log.Printf("\n🔄 Creating PR for %d successful test generations...", successCount)
			if err := createPRFromResults(ctx, results, config); err != nil {
				log.Printf("❌ Failed to create PR: %v", err)
			}
		}
	} else if successCount > 0 {
		log.Printf("\n💡 To create PR automatically, use:")
		log.Printf("   --create-pr --github-token=<token> --repo-owner=<owner> --repo-name=<repo>")
	}

	// Step 6: Cleanup temporary files
	validator.CleanupTempFiles()

	// Step 7: Print final summary
	printFinalSummary(results, successCount, failureCount)
}

// createPRFromResults creates a PR with the successful test results
func createPRFromResults(ctx context.Context, results []ProcessResult, config *Config) error {
	// Filter successful results
	var successfulResults []ProcessResult
	for _, result := range results {
		if result.Success {
			successfulResults = append(successfulResults, result)
		}
	}

	if len(successfulResults) == 0 {
		return fmt.Errorf("no successful results to create PR for")
	}

	// Create PR using the simplified PR creator
	prCreator := NewSimplifiedPRCreator(config.GitHubToken, config.RepoOwner, config.RepoName)
	
	prNumber, err := prCreator.CreateTestsPR(ctx, successfulResults)
	if err != nil {
		return fmt.Errorf("failed to create PR: %v", err)
	}

	log.Printf("🎉 Successfully created PR #%d", prNumber)
	log.Printf("🔗 View PR: https://github.com/%s/%s/pull/%d", config.RepoOwner, config.RepoName, prNumber)
	
	return nil
}

// boolToStatus converts boolean to visual status
func boolToStatus(b bool) string {
	if b {
		return "✅ Set"
	}
	return "❌ Missing"
}

// processFileComplete handles the complete workflow for a single file
// SIMPLIFIED: Read whole file → Send to LLM → Validate
func processFileComplete(ctx context.Context, sourceFile string, config *Config,
	generator *KubeEdgeTestGenerator, validator *TestValidator) ProcessResult {

	startTime := time.Now()
	result := ProcessResult{
		SourceFile: sourceFile,
		Duration:   0,
	}

	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// Generate test file path
	testFile := validator.GenerateTestFilePath(sourceFile)
	result.TestFile = testFile

	// Check if test file already exists
	if fileExists(testFile) {
		log.Printf("ℹ️ Test file already exists: %s", testFile)
		validationResult := validator.ValidateGeneratedTest(ctx, sourceFile, testFile)
		if validationResult.Success {
			result.Success = true
			result.BeforeCoverage = validationResult.BeforeCoverage
			result.AfterCoverage = validationResult.AfterCoverage
			return result
		} else {
			log.Printf("⚠️ Existing test file failed validation, regenerating...")
			os.Remove(testFile)
		}
	}

	// ===== SIMPLIFIED APPROACH: Read whole file and send to LLM =====
	log.Printf("📖 Reading source file: %s", sourceFile)
	
	sourceContent, err := os.ReadFile(sourceFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to read source file: %v", err)
		return result
	}
	
	log.Printf("📊 Source file size: %d bytes", len(sourceContent))
	
	// Basic check - ensure it's a Go file with functions
	if !hasTestableContent(string(sourceContent)) {
		result.Error = fmt.Errorf("file doesn't contain testable Go functions")
		return result
	}

	log.Printf("✅ File has testable content, proceeding with test generation")

	// Generate tests for the entire file - let LLM decide everything
	testContent, success := generateTestsWithLLMDecision(ctx, sourceFile, string(sourceContent), generator, config.MaxRetries)
	if !success {
		result.Error = fmt.Errorf("test generation failed after %d attempts", config.MaxRetries)
		return result
	}

	// Write test file
	if err := writeTestFile(testFile, testContent); err != nil {
		result.Error = fmt.Errorf("failed to write test file: %v", err)
		return result
	}

	log.Printf("📝 Generated test file: %s", testFile)

	// Validate - compile and test coverage
	validationResult := validator.ValidateGeneratedTest(ctx, sourceFile, testFile)

	if !validationResult.Success {
		os.Remove(testFile)
		result.Error = fmt.Errorf("validation failed: %v", validationResult.Error)
		return result
	}

	result.Success = true
	result.BeforeCoverage = validationResult.BeforeCoverage
	result.AfterCoverage = validationResult.AfterCoverage

	return result
}

// parseFlags parses command line arguments with proper defaults
func parseFlags() *Config {
	config := &Config{}

	flag.Float64Var(&config.CoverageThreshold, "coverage-threshold", 40.0, "Coverage threshold percentage")
	flag.IntVar(&config.MaxRetries, "max-retries", 3, "Maximum retry attempts for test generation")
	flag.StringVar(&config.GeminiAPIKey, "gemini-api-key", "", "Gemini API key for test generation")
	flag.StringVar(&config.ChangedFiles, "changed-files", "", "Comma-separated list of changed files")
	flag.StringVar(&config.WorkingDir, "working-dir", ".", "Working directory")
	flag.BoolVar(&config.Debug, "debug", false, "Enable debug logging")
	
	// PR Creation flags
	flag.BoolVar(&config.CreatePR, "create-pr", false, "Create GitHub PR with generated tests")
	flag.StringVar(&config.GitHubToken, "github-token", "", "GitHub token for PR creation")
	flag.StringVar(&config.RepoOwner, "repo-owner", "", "GitHub repository owner")
	flag.StringVar(&config.RepoName, "repo-name", "", "GitHub repository name")

	flag.Parse()

	// Get API key from environment if not provided
	if config.GeminiAPIKey == "" {
		config.GeminiAPIKey = os.Getenv("GEMINI_API_KEY")
	}

	if config.GeminiAPIKey == "" {
		log.Fatal("❌ GEMINI_API_KEY is required")
	}

	// Get GitHub token from environment if not provided
	if config.GitHubToken == "" {
		config.GitHubToken = os.Getenv("GITHUB_TOKEN")
	}

	return config
}

// parseChangedFiles parses comma-separated file list
func parseChangedFiles(filesStr string) []string {
	var files []string
	for _, file := range strings.Split(filesStr, ",") {
		file = strings.TrimSpace(file)
		if file != "" && strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, "_test.go") {
			files = append(files, file)
		}
	}
	return files
}

// generateWorkflowOutput creates output files for GitHub workflow
func generateWorkflowOutput(results []ProcessResult) {
	var successfulTests []string
	var failedTests []string

	for _, result := range results {
		if result.Success {
			line := fmt.Sprintf("%s|%s|%.2f|%.2f",
				result.SourceFile, result.TestFile, result.BeforeCoverage, result.AfterCoverage)
			successfulTests = append(successfulTests, line)
		} else {
			line := fmt.Sprintf("%s|%v", result.SourceFile, result.Error)
			failedTests = append(failedTests, line)
		}
	}

	// Write successful tests file
	if len(successfulTests) > 0 {
		content := strings.Join(successfulTests, "\n")
		if err := os.WriteFile("successful_tests.txt", []byte(content), 0644); err != nil {
			log.Printf("⚠️ Warning: Failed to write successful_tests.txt: %v", err)
		}
	}

	// Write failed tests file
	if len(failedTests) > 0 {
		content := strings.Join(failedTests, "\n")
		if err := os.WriteFile("failed_tests.txt", []byte(content), 0644); err != nil {
			log.Printf("⚠️ Warning: Failed to write failed_tests.txt: %v", err)
		}
	}
}

// printCoverageSummary prints coverage summary for all checked files
func printCoverageSummary(coverageInfos []CoverageInfo, threshold float64) {
	fmt.Println("\n📊 COVERAGE SUMMARY")
	fmt.Println(strings.Repeat("=", 50))
	for _, info := range coverageInfos {
		status := "✅ SUFFICIENT"
		if info.Coverage < threshold {
			status = "🎯 NEEDS TESTS"
		}
		fmt.Printf("%-30s %.2f%% %s\n", info.FilePath, info.Coverage, status)
	}
}

// printFinalSummary prints the final processing summary
func printFinalSummary(results []ProcessResult, successCount, failureCount int) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🏁 KUBEEDGE AUTO TEST GENERATOR - FINAL SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("📈 Total Processed: %d\n", len(results))
	fmt.Printf("✅ Successful: %d\n", successCount)
	fmt.Printf("❌ Failed: %d\n", failureCount)

	if successCount > 0 {
		fmt.Println("\n🎉 SUCCESSFUL GENERATIONS:")
		for _, result := range results {
			if result.Success {
				improvement := result.AfterCoverage - result.BeforeCoverage
				fmt.Printf("  📁 %-40s %.2f%% → %.2f%% (+%.2f%%)\n",
					result.SourceFile, result.BeforeCoverage, result.AfterCoverage, improvement)
			}
		}
	}

	if failureCount > 0 {
		fmt.Println("\n💥 FAILED GENERATIONS:")
		for _, result := range results {
			if !result.Success {
				fmt.Printf("  📁 %-40s %v\n", result.SourceFile, result.Error)
			}
		}
	}

	fmt.Println(strings.Repeat("=", 60))

	if successCount > 0 {
		fmt.Println("🚀 Ready for PR creation! Check successful_tests.txt for details.")
	}
}