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
	"regexp"
	"strings"
)

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// writeTestFile writes test content to a file
func writeTestFile(testFile, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(testFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	// Write file
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %v", testFile, err)
	}

	return nil
}

// hasTestableContent checks if the file has content worth testing
func hasTestableContent(content string) bool {
	// Basic checks for Go file with functions
	if !strings.Contains(content, "package ") {
		log.Printf("❌ No package declaration found")
		return false
	}
	
	if !strings.Contains(content, "func ") {
		log.Printf("❌ No functions found")
		return false
	}
	
	// Count exportable functions (not main, init, or test functions)
	funcCount := 0
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "func ") && 
		   !strings.Contains(line, "func main") && 
		   !strings.Contains(line, "func init") && 
		   !strings.Contains(line, "func Test") &&
		   !strings.Contains(line, "func Benchmark") &&
		   !strings.Contains(line, "func Example") {
			funcCount++
		}
	}
	
	log.Printf("📊 Found %d potential functions to test", funcCount)
	return funcCount > 0
}

// generateTestsWithLLMDecision - let LLM decide everything
func generateTestsWithLLMDecision(ctx context.Context, filePath string, sourceContent string, 
	generator *KubeEdgeTestGenerator, maxRetries int, workingDir string) (string, bool) {
	
	var lastError error
	var lastGeneratedContent string

	// Check if existing test file exists
	testFilePath := generateTestFilePath(filePath)
	absTestFilePath := filepath.Join(workingDir, testFilePath)
	
	var existingTestContent string
	if fileExists(absTestFilePath) {
		log.Printf("📖 Found existing test file: %s", testFilePath)
		content, err := os.ReadFile(absTestFilePath)
		if err != nil {
			log.Printf("⚠️ Warning: Could not read existing test file: %v", err)
			existingTestContent = ""
		} else {
			existingTestContent = string(content)
			log.Printf("📊 Existing test file size: %d bytes", len(existingTestContent))
		}
	} else {
		log.Printf("📝 No existing test file found, generating new one")
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if existingTestContent != "" {
			log.Printf("🤖 Attempt %d/%d: Enhancing existing test file for %s", attempt, maxRetries, filePath)
		} else {
			log.Printf("🤖 Attempt %d/%d: Generating new test file for %s", attempt, maxRetries, filePath)
		}
		
		testContent, err := generator.GenerateTestsFromWholeFile(ctx, filePath, sourceContent, existingTestContent, lastError)
		if err != nil {
			lastError = err
			log.Printf("❌ Attempt %d failed: %v", attempt, err)
			continue
		}

		// Save the generated content for debugging
		lastGeneratedContent = testContent

		// Basic validation - check if it looks like Go test code
		if isValidGoTestContent(testContent) {
			log.Printf("✅ Test generation successful on attempt %d", attempt)
			return testContent, true
		}

		lastError = fmt.Errorf("generated content doesn't look like valid Go test code")
		log.Printf("⚠️ Attempt %d produced insufficient content", attempt)
	}

	log.Printf("❌ All %d attempts failed for %s", maxRetries, filePath)
	// Return the last generated content even if it failed, for debugging
	return lastGeneratedContent, false
}

// generateTestFilePath generates the test file path for a source file
func generateTestFilePath(sourceFile string) string {
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(dir, name+"_test.go")
}

// isValidGoTestContent - simplified validation
func isValidGoTestContent(content string) bool {
	required := []string{"package ", "import", "func Test", "testing"}
	for _, req := range required {
		if !strings.Contains(content, req) {
			log.Printf("⚠️ Generated content missing: %s", req)
			return false
		}
	}
	return true
}

// cleanupGoCode removes unused imports and formats the Go code
func cleanupGoCode(filePath string) error {
	log.Printf("🧹 Cleaning up unused imports in: %s", filePath)
	
	// Try goimports first (preferred - handles imports automatically)
	if err := runGoImports(filePath); err == nil {
		log.Printf("✅ Successfully cleaned with goimports")
		return nil
	}
	
	// Fallback to gofmt if goimports not available
	if err := runGoFmt(filePath); err == nil {
		log.Printf("✅ Successfully formatted with gofmt")
		return nil
	}
	
	log.Printf("⚠️ Could not clean up imports, but file may still be valid")
	return nil
}

// runGoImports runs goimports to fix imports and format code
func runGoImports(filePath string) error {
	// Try goimports from various locations
	goimportsPaths := []string{
		"goimports",                    // if it's in PATH
		os.ExpandEnv("$HOME/go/bin/goimports"), // default Go bin location
		"/usr/local/go/bin/goimports",          // system Go installation
	}
	
	for _, goimportsPath := range goimportsPaths {
		cmd := exec.Command(goimportsPath, "-w", filePath)
		_, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		// Continue to next path if this one failed
	}
	
	return fmt.Errorf("goimports not found in any expected location")
}

// runGoFmt runs gofmt to format code (doesn't fix imports but formats)
func runGoFmt(filePath string) error {
	cmd := exec.Command("gofmt", "-w", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("🔍 gofmt output: %s", string(output))
		return fmt.Errorf("gofmt failed: %v", err)
	}
	return nil
}

// parseFailingTestFunctions extracts failing test function names from test output
func parseFailingTestFunctions(testOutput string) []string {
	var failingTests []string
	
	// Pattern to match: "--- FAIL: TestFunctionName (0.00s)"
	failPattern := regexp.MustCompile(`--- FAIL: (Test\w+) \(`)
	matches := failPattern.FindAllStringSubmatch(testOutput, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			testName := match[1]
			failingTests = append(failingTests, testName)
			log.Printf("🔍 Found failing test: %s", testName)
		}
	}
	
	return failingTests
}

// removeFailingTestFunctions removes specific test functions from a Go test file
func removeFailingTestFunctions(filePath string, failingTests []string) error {
	if len(failingTests) == 0 {
		return nil // Nothing to remove
	}
	
	log.Printf("🧹 Removing %d failing test functions from %s", len(failingTests), filePath)
	
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}
	
	fileContent := string(content)
	
	// Remove each failing test function
	for _, testName := range failingTests {
		log.Printf("🗑️ Removing test function: %s", testName)
		fileContent = removeTestFunction(fileContent, testName)
	}
	
	// Write the modified content back
	err = os.WriteFile(filePath, []byte(fileContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write modified file: %v", err)
	}
	
	log.Printf("✅ Successfully removed failing tests, keeping remaining tests")
	return nil
}

// removeTestFunction removes a specific test function from Go source code
func removeTestFunction(content string, testName string) string {
	// For functions with nested braces, we use a sophisticated approach
	return removeComplexTestFunction(content, testName)
}

// removeComplexTestFunction handles test functions with nested braces
func removeComplexTestFunction(content string, testName string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inTargetFunction := false
	braceCount := 0
	hasSeenOpenBrace := false
	
	funcPattern := regexp.MustCompile(`^\s*func\s+` + regexp.QuoteMeta(testName) + `\s*\(`)
	
	for i, line := range lines {
		if !inTargetFunction {
			// Look for the start of our target function
			if funcPattern.MatchString(line) {
				inTargetFunction = true
				hasSeenOpenBrace = false
				braceCount = 0
				log.Printf("🔍 Found start of %s at line %d: %s", testName, i+1, strings.TrimSpace(line))
				
				// Count braces in the function declaration line
				for _, char := range line {
					if char == '{' {
						braceCount++
						hasSeenOpenBrace = true
					} else if char == '}' {
						braceCount--
					}
				}
				
				// If function is on one line, we're done
				if hasSeenOpenBrace && braceCount == 0 {
					inTargetFunction = false
					log.Printf("🔍 Single-line function %s at line %d", testName, i+1)
				}
				continue // Skip this line
			}
			result = append(result, line)
		} else {
			// We're inside the target function, count braces
			for _, char := range line {
				if char == '{' {
					braceCount++
					hasSeenOpenBrace = true
				} else if char == '}' {
					braceCount--
				}
			}
			
			// If braces are balanced and we've seen at least one opening brace, function is complete
			if hasSeenOpenBrace && braceCount == 0 {
				inTargetFunction = false
				log.Printf("🔍 Found end of %s at line %d: %s", testName, i+1, strings.TrimSpace(line))
				continue // Skip this line too
			}
			// Skip all lines while we're inside the target function
		}
	}
	
	// Clean up excessive empty lines
	cleanedResult := cleanupEmptyLines(result)
	
	return strings.Join(cleanedResult, "\n")
}

// cleanupEmptyLines removes excessive consecutive empty lines
func cleanupEmptyLines(lines []string) []string {
	var result []string
	emptyLineCount := 0
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyLineCount++
			// Keep at most 2 consecutive empty lines
			if emptyLineCount <= 2 {
				result = append(result, line)
			}
		} else {
			emptyLineCount = 0
			result = append(result, line)
		}
	}
	
	return result
}