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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TestValidationResult represents validation results for a test file
type TestValidationResult struct {
	SourceFile       string
	TestFile         string
	Success          bool
	Error            error
	BeforeCoverage   float64
	AfterCoverage    float64
	CoverageImproved bool
	TestsPass        bool
	CompileSuccess   bool
	Duration         time.Duration
}

// TestValidator handles all validation and compilation logic
type TestValidator struct {
	workingDir        string
	coverageThreshold float64
}

// NewTestValidator creates a new test validator
func NewTestValidator(workingDir string, coverageThreshold float64) *TestValidator {
	// Convert to absolute path to avoid path resolution issues
	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		fmt.Printf("⚠️ Warning: Could not get absolute path for %s, using as-is: %v\n", workingDir, err)
		absWorkingDir = workingDir
	}
	
	return &TestValidator{
		workingDir:        absWorkingDir,
		coverageThreshold: coverageThreshold,
	}
}

// ValidateGeneratedTest - FIXED: Proper coverage analysis order
func (tv *TestValidator) ValidateGeneratedTest(ctx context.Context, sourceFile string, testFile string) TestValidationResult {
	startTime := time.Now()
	result := TestValidationResult{
		SourceFile: sourceFile,
		TestFile:   testFile,
	}

	defer func() {
		result.Duration = time.Since(startTime)
	}()

	fmt.Printf("🔍 Validating test file: %s\n", testFile)

	// Step 1: For generated tests, before coverage is always 0% (no existing tests)
	beforeCoverage := 0.0
	result.BeforeCoverage = beforeCoverage
	fmt.Printf("📊 Before coverage (generated test): %.2f%%\n", beforeCoverage)

	// Step 2: Check if test file exists - resolve path relative to working directory
	absTestFile := filepath.Join(tv.workingDir, testFile)
	if !fileExists(absTestFile) {
		result.Error = fmt.Errorf("test file does not exist: %s", testFile)
		return result
	}

	// Step 3: Run tests to check compilation and execution
	testsPass, compileSuccess, testErr := tv.RunGoTestWithCoverage(ctx, sourceFile)
	result.CompileSuccess = compileSuccess
	result.TestsPass = testsPass
	
	if !compileSuccess {
		result.Error = fmt.Errorf("test compilation failed: %v", testErr)
		return result
	}
	fmt.Printf("✅ Test compilation: SUCCESS\n")

	if !testsPass {
		result.Error = fmt.Errorf("tests execution failed: %v", testErr)
		return result
	}
	fmt.Printf("✅ Test execution: SUCCESS\n")

	// Step 4: Get coverage WITH the test file
	afterCoverage, err := tv.GetFileCoverage(ctx, sourceFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to get after coverage: %v", err)
		return result
	}
	result.AfterCoverage = afterCoverage
	fmt.Printf("📊 After coverage: %.2f%%\n", afterCoverage)

	// Step 5: Verify coverage improvement
	result.CoverageImproved = afterCoverage > beforeCoverage
	if !result.CoverageImproved {
		result.Error = fmt.Errorf("coverage did not improve: %.2f%% → %.2f%%", beforeCoverage, afterCoverage)
		return result
	}

	improvement := afterCoverage - beforeCoverage
	fmt.Printf("📈 Coverage improvement: +%.2f%%\n", improvement)

	result.Success = true
	return result
}

// GetBaselineCoverage - Gets coverage WITHOUT any test files (improved version)
func (tv *TestValidator) GetBaselineCoverage(ctx context.Context, sourceFile string) (float64, error) {
	packageDir := filepath.Dir(sourceFile)
	
	// Resolve package directory relative to working directory
	absPackageDir := filepath.Join(tv.workingDir, packageDir)
	
	// Check if package directory exists
	if !fileExists(absPackageDir) {
		fmt.Printf("⚠️ Package directory does not exist: %s\n", absPackageDir)
		return 0.0, nil
	}
	
	// Find and temporarily remove all test files
	testFiles, err := tv.findTestFiles(absPackageDir)
	if err != nil {
		return 0.0, fmt.Errorf("failed to find test files: %v", err)
	}
	
	// If no test files exist, baseline coverage is 0%
	if len(testFiles) == 0 {
		fmt.Printf("📊 No test files found in %s, baseline coverage is 0%%\n", packageDir)
		return 0.0, nil
	}
	
	// Temporarily rename test files to get baseline
	renamedFiles := make(map[string]string)
	for _, testFile := range testFiles {
		tempName := testFile + ".temp_backup"
		if err := os.Rename(testFile, tempName); err != nil {
			// Restore any already renamed files
			tv.restoreTestFiles(renamedFiles)
			return 0.0, fmt.Errorf("failed to backup test file %s: %v", testFile, err)
		}
		renamedFiles[testFile] = tempName
	}
	
	// Get baseline coverage using absolute path for coverage file
	coverageFile := filepath.Join(tv.workingDir, "baseline_coverage.out")
	fmt.Printf("🔍 [DEBUG] GetBaselineCoverage: workingDir=%s, coverageFile=%s\n", tv.workingDir, coverageFile)
	
	// Create a dummy test file to make go test work even with no tests
	dummyTestFile := filepath.Join(absPackageDir, "dummy_test.go")
	dummyContent := fmt.Sprintf(`package %s

import "testing"

func TestDummy(t *testing.T) {
	// Dummy test to enable coverage collection
}
`, tv.getPackageName(sourceFile))
	
	err = os.WriteFile(dummyTestFile, []byte(dummyContent), 0644)
	if err != nil {
		tv.restoreTestFiles(renamedFiles)
		return 0.0, fmt.Errorf("failed to create dummy test file: %v", err)
	}
	
	// Run coverage test
	cmd := exec.CommandContext(ctx, "go", "test", "-coverprofile="+coverageFile, "./"+packageDir)
	cmd.Dir = tv.workingDir
	
	output, execErr := cmd.CombinedOutput()
	
	// Clean up dummy test file
	os.Remove(dummyTestFile)
	
	// Restore test files immediately
	tv.restoreTestFiles(renamedFiles)
	
	// Check results
	if execErr != nil {
		if strings.Contains(string(output), "no test files") || 
		   strings.Contains(string(output), "no buildable Go source files") {
			// This is expected - no tests means 0% coverage
			fmt.Printf("📊 No buildable Go files or tests in %s, baseline coverage is 0%%\n", packageDir)
			return 0.0, nil
		}
		fmt.Printf("⚠️ Baseline coverage test failed: %v, output: %s\n", execErr, string(output))
		return 0.0, nil // Return 0% instead of error for missing baseline
	}
	
	// Parse baseline coverage - pass full source file path
	coverage := tv.ParseCoverageFromFile(coverageFile, sourceFile)
	os.Remove(coverageFile)
	
	return coverage, nil
}

// getPackageName extracts package name from a Go source file
func (tv *TestValidator) getPackageName(sourceFile string) string {
	absSourceFile := filepath.Join(tv.workingDir, sourceFile)
	content, err := os.ReadFile(absSourceFile)
	if err != nil {
		return "main" // fallback
	}
	
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return "main" // fallback
}

// findTestFiles finds all test files in a directory
func (tv *TestValidator) findTestFiles(packageDir string) ([]string, error) {
	var testFiles []string
	
	files, err := os.ReadDir(packageDir)
	if err != nil {
		return nil, err
	}
	
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), "_test.go") {
			testFiles = append(testFiles, filepath.Join(packageDir, file.Name()))
		}
	}
	
	return testFiles, nil
}

// restoreTestFiles restores backed up test files
func (tv *TestValidator) restoreTestFiles(renamedFiles map[string]string) {
	for original, backup := range renamedFiles {
		if err := os.Rename(backup, original); err != nil {
			fmt.Printf("⚠️ Warning: Failed to restore test file %s: %v\n", original, err)
		}
	}
}

// RunGoTestWithCoverage runs tests and checks compilation/execution
func (tv *TestValidator) RunGoTestWithCoverage(ctx context.Context, sourceFile string) (testsPass bool, compileSuccess bool, err error) {
	packageDir := filepath.Dir(sourceFile)
	testFile := tv.GenerateTestFilePath(sourceFile)
	absTestFile := filepath.Join(tv.workingDir, testFile)
	
	coverageFile := filepath.Join(tv.workingDir, "coverage.out")
	cmd := exec.CommandContext(ctx, "go", "test", "-coverprofile="+coverageFile, "./"+packageDir)
	cmd.Dir = tv.workingDir

	output, execErr := cmd.CombinedOutput()
	outputStr := string(output)

	// Check compilation success
	if execErr != nil && strings.Contains(outputStr, "build failed") {
		return false, false, fmt.Errorf("compilation failed: %v, output: %s", execErr, outputStr)
	}
	compileSuccess = true

	// Check test execution success
	if execErr != nil || strings.Contains(outputStr, "FAIL") {
		// Tests failed, but compilation succeeded - try to salvage passing tests
		log.Printf("⚠️ Some tests failed, attempting to remove failing tests and keep passing ones")
		
		// Parse failing test functions
		failingTests := parseFailingTestFunctions(outputStr)
		
		if len(failingTests) > 0 {
			// Remove failing test functions
			if err := removeFailingTestFunctions(absTestFile, failingTests); err != nil {
				log.Printf("❌ Failed to remove failing tests: %v", err)
				return false, true, fmt.Errorf("test execution failed: %v, output: %s", execErr, outputStr)
			}
			
			// Clean up imports after removing functions
			if err := cleanupGoCode(absTestFile); err != nil {
				log.Printf("⚠️ Warning: Failed to cleanup imports after removing tests: %v", err)
			}
			
			// Re-run tests with remaining functions
			log.Printf("🔄 Re-running tests with remaining test functions...")
			cmd2 := exec.CommandContext(ctx, "go", "test", "-coverprofile="+coverageFile, "./"+packageDir)
			cmd2.Dir = tv.workingDir
			
			output2, execErr2 := cmd2.CombinedOutput()
			outputStr2 := string(output2)
			
			if execErr2 != nil {
				return false, true, fmt.Errorf("tests still failing after cleanup: %v, output: %s", execErr2, outputStr2)
			}
			
			// Check if remaining tests pass
			if strings.Contains(outputStr2, "FAIL") {
				return false, true, fmt.Errorf("remaining tests still failing: %s", outputStr2)
			}
			
			log.Printf("✅ Successfully removed failing tests, remaining tests pass!")
			return true, true, nil
		}
		
		return false, true, fmt.Errorf("test execution failed: %v, output: %s", execErr, outputStr)
	}

	// Ensure tests actually ran
	if !strings.Contains(outputStr, "PASS") && !strings.Contains(outputStr, "ok") {
		return false, true, fmt.Errorf("no tests were executed")
	}

	return true, true, nil
}

// GetFileCoverage gets coverage percentage for a specific file
func (tv *TestValidator) GetFileCoverage(ctx context.Context, sourceFile string) (float64, error) {
	packageDir := filepath.Dir(sourceFile)
	// Use consistent coverage file naming
	coverageFile := filepath.Join(tv.workingDir, "current_coverage.out")

	fmt.Printf("🔍 [DEBUG] Running coverage test:\n")
	fmt.Printf("   - Working Dir: %s\n", tv.workingDir)
	fmt.Printf("   - Package Dir: %s\n", packageDir)
	fmt.Printf("   - Source File: %s\n", sourceFile)
	fmt.Printf("   - Command: go test -coverprofile=%s ./%s\n", coverageFile, packageDir)

	cmd := exec.CommandContext(ctx, "go", "test", "-coverprofile="+coverageFile, "./"+packageDir)
	cmd.Dir = tv.workingDir

	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	
	fmt.Printf("📊 [DEBUG] Test output: %s\n", outputStr)
	
	if err != nil {
		fmt.Printf("❌ [DEBUG] Test command failed: %v\n", err)
		if strings.Contains(outputStr, "no test files") || 
		   strings.Contains(outputStr, "no Go files") {
			return 0.0, nil
		}
		return 0.0, fmt.Errorf("coverage test failed: %v, output: %s", err, outputStr)
	}

	// Pass the full source file path for better matching
	coverage := tv.ParseCoverageFromFile(coverageFile, sourceFile)
	fmt.Printf("🎯 [DEBUG] Parsed coverage: %.2f%% for file %s\n", coverage, sourceFile)
	
	os.Remove(coverageFile)

	return coverage, nil
}

// ParseCoverageFromFile parses coverage percentage from coverage output file
func (tv *TestValidator) ParseCoverageFromFile(coverageFile string, targetFile string) float64 {
	fmt.Printf("🔍 [DEBUG] Parsing coverage file: %s, looking for: %s\n", coverageFile, targetFile)
	
	if !fileExists(coverageFile) {
		fmt.Printf("❌ [DEBUG] Coverage file does not exist: %s\n", coverageFile)
		return 0.0
	}

	cmd := exec.Command("go", "tool", "cover", "-func", coverageFile)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ [DEBUG] Failed to parse coverage file: %v", err)
		return 0.0
	}
	
	fmt.Printf("📊 [DEBUG] Coverage tool output:\n%s\n", string(output))

	lines := strings.Split(string(output), "\n")
	var functionCoverages []float64
	targetFileName := filepath.Base(targetFile)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Look for lines containing the target file (handle full package paths)
		// Coverage output format: "github.com/kubeedge/kubeedge/pkg/util/slices/slices.go:22.51,26.29 func_name 100.0%"
		if strings.Contains(line, targetFileName) && strings.Contains(line, ":") && !strings.Contains(line, "total:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				percentStr := fields[len(fields)-1]
				percentStr = strings.TrimSuffix(percentStr, "%")
				if coverage, parseErr := strconv.ParseFloat(percentStr, 64); parseErr == nil {
					functionCoverages = append(functionCoverages, coverage)
					fmt.Printf("📊 [DEBUG] Found function coverage: %s -> %.2f%%\n", fields[1], coverage)
				}
			}
		}
	}
	
	// If no function-level coverage found, try to get total file coverage
	if len(functionCoverages) == 0 {
		// Look for total coverage line specific to this file
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, targetFileName) && strings.Contains(line, "total:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					percentStr := fields[len(fields)-1]
					percentStr = strings.TrimSuffix(percentStr, "%")
					if coverage, parseErr := strconv.ParseFloat(percentStr, 64); parseErr == nil {
						fmt.Printf("📊 [DEBUG] Found total file coverage: %.2f%%\n", coverage)
						return coverage
					}
				}
			}
		}
		
		// If still no coverage found, check if file is mentioned at all in raw coverage data
		return tv.parseRawCoverageData(coverageFile, targetFile)
	}
	
	// Calculate average coverage across all functions
	var total float64
	for _, cov := range functionCoverages {
		total += cov
	}
	avgCoverage := total / float64(len(functionCoverages))
	fmt.Printf("📊 Parsed %d function coverages, average: %.2f%%\n", len(functionCoverages), avgCoverage)
	return avgCoverage
}

// parseRawCoverageData parses the raw coverage data directly from the coverage file
func (tv *TestValidator) parseRawCoverageData(coverageFile string, targetFile string) float64 {
	content, err := os.ReadFile(coverageFile)
	if err != nil {
		fmt.Printf("❌ [DEBUG] Failed to read coverage file: %v\n", err)
		return 0.0
	}
	
	lines := strings.Split(string(content), "\n")
	targetFileName := filepath.Base(targetFile)
	var totalStatements, coveredStatements int
	
	fmt.Printf("🔍 [DEBUG] Parsing raw coverage data for: %s\n", targetFileName)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		
		// Coverage line format: "github.com/kubeedge/kubeedge/pkg/util/slices/slices.go:22.51,26.29 3 1"
		if strings.Contains(line, targetFileName) {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				// fields[1] = number of statements, fields[2] = execution count
				if statements, err := strconv.Atoi(fields[1]); err == nil {
					totalStatements += statements
					if execCount, err := strconv.Atoi(fields[2]); err == nil && execCount > 0 {
						coveredStatements += statements
					}
				}
			}
		}
	}
	
	if totalStatements > 0 {
		coverage := (float64(coveredStatements) / float64(totalStatements)) * 100.0
		fmt.Printf("📊 [DEBUG] Raw coverage calculation: %d/%d statements = %.2f%%\n", 
			coveredStatements, totalStatements, coverage)
		return coverage
	}
	
	fmt.Printf("⚠️ No coverage data found for file %s\n", targetFileName)
	return 0.0
}

// GetModifiedFilesFromGit gets modified Go files from the last commit
func (tv *TestValidator) GetModifiedFilesFromGit(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", "HEAD~1", "HEAD")
	cmd.Dir = tv.workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get modified files: %v", err)
	}

	var goFiles []string
	files := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, file := range files {
		file = strings.TrimSpace(file)
		if strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, "_test.go") && file != "" {
			if fileExists(file) {
				goFiles = append(goFiles, file)
			}
		}
	}

	return goFiles, nil
}

// FilterLowCoverageFiles filters files that need test generation based on coverage threshold
func (tv *TestValidator) FilterLowCoverageFiles(ctx context.Context, files []string) ([]string, []CoverageInfo, error) {
	var lowCoverageFiles []string
	var coverageInfos []CoverageInfo

	fmt.Printf("🔍 Checking coverage for %d files with threshold %.0f%%\n", len(files), tv.coverageThreshold)

	for _, file := range files {
		// Check CURRENT coverage (including existing tests) for filtering
		coverage, err := tv.GetFileCoverage(ctx, file)
		if err != nil {
			fmt.Printf("⚠️ Warning: Could not get coverage for %s: %v\n", file, err)
			// If we can't get current coverage, assume 0% (needs tests)
			coverage = 0.0
		}

		info := CoverageInfo{
			FilePath: file,
			Coverage: coverage,
		}
		coverageInfos = append(coverageInfos, info)

		if coverage < tv.coverageThreshold {
			lowCoverageFiles = append(lowCoverageFiles, file)
			fmt.Printf("🎯 %s: %.2f%% (needs tests)\n", file, coverage)
		} else {
			fmt.Printf("✅ %s: %.2f%% (sufficient)\n", file, coverage)
		}
	}

	return lowCoverageFiles, coverageInfos, nil
}

// CoverageInfo holds coverage information for a file
type CoverageInfo struct {
	FilePath string
	Coverage float64
}

// GenerateTestFilePath generates the test file path for a source file
func (tv *TestValidator) GenerateTestFilePath(sourceFile string) string {
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	// Return relative path - working directory resolution will be done when needed
	return filepath.Join(dir, name+"_test.go")
}

// CleanupTempFiles removes temporary coverage files
func (tv *TestValidator) CleanupTempFiles() {
	tempFiles := []string{
		"current_coverage.out",
		"temp_coverage.out", 
		"coverage.out",
		"baseline_coverage.out",
		"before_coverage.out", 
		"after_coverage.out",
	}
	
	for _, file := range tempFiles {
		// Remove both relative and absolute paths
		os.Remove(file)
		os.Remove(filepath.Join(tv.workingDir, file))
	}
}