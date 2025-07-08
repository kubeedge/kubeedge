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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type KubeEdgeTestGenerator struct {
	client    *genai.Client
	templates map[string]string
}

func NewKubeEdgeTestGenerator(apiKey string) *KubeEdgeTestGenerator {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		panic(fmt.Sprintf("Failed to create Gemini client: %v", err))
	}

	templates := map[string]string{
		"gomonkey":  loadGoMonkeyTemplate(),
		"ginkgo":    loadGinkgoTemplate(),
		"standard":  loadStandardTemplate(),
	}

	return &KubeEdgeTestGenerator{
		client:    client,
		templates: templates,
	}
}

// GenerateTestsFromWholeFile - NEW METHOD: Let LLM decide everything about testing approach
func (ktg *KubeEdgeTestGenerator) GenerateTestsFromWholeFile(ctx context.Context, filePath string, sourceContent string, previousError error) (string, error) {
	
	// Extract package name for proper test file structure
	packageName := ktg.extractPackageName(sourceContent)
	
	// Build ultra-simple prompt - let LLM decide everything
	prompt := ktg.buildSimplePrompt(filePath, sourceContent, previousError)

	// Generate with Gemini AI
	testContent, err := ktg.generateWithGemini(ctx, prompt, "auto")
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	// Minimal cleanup - just ensure package name is correct
	finalTestContent := ktg.ensureCorrectPackage(testContent, packageName)

	return finalTestContent, nil
}

// buildSimplePrompt - ultra-simple prompt that lets LLM decide everything
func (ktg *KubeEdgeTestGenerator) buildSimplePrompt(filePath string, sourceContent string, previousError error) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert Go test generator for KubeEdge project.\n\n")
	
	if previousError != nil {
		prompt.WriteString("PREVIOUS ATTEMPT FAILED:\n")
		prompt.WriteString(fmt.Sprintf("Error: %v\n", previousError))
		prompt.WriteString("Please fix the issues and generate working code.\n\n")
	}

	prompt.WriteString("TASK: Generate comprehensive unit tests for this Go file.\n\n")

	prompt.WriteString("REQUIREMENTS:\n")
	prompt.WriteString("1. Analyze the code and decide what testing approach to use\n")
	prompt.WriteString("2. Choose appropriate imports (standard testing, testify, gomonkey if needed)\n")
	prompt.WriteString("3. If mocking is needed, use: github.com/agiledragon/gomonkey/v2 (NOT github.com/agtorre/go-gomonkey/v2)\n")
	prompt.WriteString("4. Create meaningful test cases for all exportable functions\n")
	prompt.WriteString("5. Include edge cases, error cases, and boundary conditions\n")
	prompt.WriteString("6. Make sure the code compiles and runs\n")
	prompt.WriteString("7. Use table-driven tests where appropriate\n\n")

	prompt.WriteString("GUIDELINES:\n")
	prompt.WriteString("- For simple functions (math, string ops): use standard testing with testify\n")
	prompt.WriteString("- For functions with external dependencies: use mocking\n")
	prompt.WriteString("- For complex business logic: use comprehensive test cases\n")
	prompt.WriteString("- Always include both positive and negative test scenarios\n")
	prompt.WriteString("- For math functions like Add, Subtract: do NOT use gomonkey, use standard testing\n\n")

	prompt.WriteString("SOURCE FILE TO TEST:\n")
	prompt.WriteString("```go\n")
	prompt.WriteString(sourceContent)
	prompt.WriteString("\n```\n\n")

	prompt.WriteString("OUTPUT: Generate ONLY the complete test file content. Start with package declaration and imports.\n")

	return prompt.String()
}

// extractPackageName - simple package name extraction
func (ktg *KubeEdgeTestGenerator) extractPackageName(content string) string {
	lines := strings.Split(content, "\n")
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

// ensureCorrectPackage - minimal cleanup to ensure package name is correct
func (ktg *KubeEdgeTestGenerator) ensureCorrectPackage(content string, expectedPackage string) string {
	// Remove markdown code blocks if present
	content = regexp.MustCompile("```go\n?").ReplaceAllString(content, "")
	content = regexp.MustCompile("```\n?").ReplaceAllString(content, "")
	content = strings.TrimSpace(content)
	
	// Ensure correct package declaration
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			lines[i] = fmt.Sprintf("package %s", expectedPackage)
			break
		}
	}
	
	return strings.Join(lines, "\n")
}

// generateWithGemini calls Gemini API to generate test content
func (ktg *KubeEdgeTestGenerator) generateWithGemini(ctx context.Context, prompt string, _ string) (string, error) {
	model := ktg.client.GenerativeModel("gemini-1.5-flash")
	
	// Configure model for code generation
	model.SetTemperature(0.3) // Lower temperature for more consistent code
	model.SetTopK(40)
	model.SetTopP(0.95)
	
	// Add timeout context (2 minutes)
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	
	startTime := time.Now()
	resp, err := model.GenerateContent(timeoutCtx, genai.Text(prompt))
	duration := time.Since(startTime)
	
	fmt.Printf("⏱️ API Call Duration: %v\n", duration)
	
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	generatedCode := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			generatedCode += string(text)
		}
	}

	if generatedCode == "" {
		return "", fmt.Errorf("empty content generated")
	}

	fmt.Printf("Successfully generated %d characters of code\n", len(generatedCode))
	return generatedCode, nil
}

// findRepoRoot finds the repository root by looking for go.mod
func (ktg *KubeEdgeTestGenerator) findRepoRoot(startDir string) string {
	currentDir := startDir
	
	for i := 0; i < 10; i++ { // Limit search to prevent infinite loop
		// Check if go.mod exists and contains kubeedge
		goModPath := filepath.Join(currentDir, "go.mod")
		if fileExists(goModPath) {
			// Read go.mod to verify it's the KubeEdge repository
			content, err := os.ReadFile(goModPath)
			if err == nil && strings.Contains(string(content), "github.com/kubeedge/kubeedge") {
				return currentDir
			}
		}
		
		// Go up one level
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached filesystem root
			break
		}
		currentDir = parentDir
	}
	
	return ""
}

// Close closes the Gemini client
func (ktg *KubeEdgeTestGenerator) Close() {
	if ktg.client != nil {
		ktg.client.Close()
	}
}

