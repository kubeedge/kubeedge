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
	"path/filepath"
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
	generator *KubeEdgeTestGenerator, maxRetries int) (string, bool) {
	
	var lastError error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("🤖 Attempt %d/%d: Generating tests for entire file %s", attempt, maxRetries, filePath)
		
		testContent, err := generator.GenerateTestsFromWholeFile(ctx, filePath, sourceContent, lastError)
		if err != nil {
			lastError = err
			log.Printf("❌ Attempt %d failed: %v", attempt, err)
			continue
		}

		// Basic validation - check if it looks like Go test code
		if isValidGoTestContent(testContent) {
			log.Printf("✅ Test generation successful on attempt %d", attempt)
			return testContent, true
		}

		lastError = fmt.Errorf("generated content doesn't look like valid Go test code")
		log.Printf("⚠️ Attempt %d produced insufficient content", attempt)
	}

	log.Printf("❌ All %d attempts failed for %s", maxRetries, filePath)
	return "", false
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