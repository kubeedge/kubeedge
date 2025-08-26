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
	"regexp"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type KubeEdgeTestGenerator struct {
	client *genai.Client
}

func NewKubeEdgeTestGenerator(apiKey string) *KubeEdgeTestGenerator {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		panic(fmt.Sprintf("Failed to create Gemini client: %v", err))
	}

	return &KubeEdgeTestGenerator{
		client: client,
	}
}

// GenerateTestsFromWholeFile - NEW METHOD: Let LLM decide everything about testing approach
func (ktg *KubeEdgeTestGenerator) GenerateTestsFromWholeFile(ctx context.Context, filePath string, sourceContent string, existingTestContent string, previousError error) (string, error) {
	
	// Extract package name for proper test file structure
	packageName := ktg.extractPackageName(sourceContent)
	
	// Build prompt based on whether existing test exists
	var prompt string
	if existingTestContent != "" {
		prompt = ktg.buildEnhancementPrompt(filePath, sourceContent, existingTestContent, previousError)
	} else {
		prompt = ktg.buildSimplePrompt(filePath, sourceContent, previousError)
	}

	// Generate with Gemini AI
	testContent, err := ktg.generateWithGemini(ctx, prompt, "auto")
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	// Minimal cleanup - just ensure package name is correct
	finalTestContent := ktg.ensureCorrectPackage(testContent, packageName)

	return finalTestContent, nil
}

// buildSimplePrompt - simplified prompt for new test file generation
func (ktg *KubeEdgeTestGenerator) buildSimplePrompt(_ string, sourceContent string, previousError error) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert Go test generator for KubeEdge project.\n\n")
	
	if previousError != nil {
		prompt.WriteString("PREVIOUS ATTEMPT FAILED:\n")
		prompt.WriteString(fmt.Sprintf("Error: %v\n", previousError))
		prompt.WriteString("Please fix the issues and generate working code.\n\n")
	}

	prompt.WriteString("TASK: Generate comprehensive unit tests for this Go file.\n\n")

	prompt.WriteString("REQUIREMENTS:\n")
	prompt.WriteString("1. Analyze the source code and create unit tests for ALL exported functions\n")
	prompt.WriteString("2. Use whatever imports are required (testing, errors, etc.)\n")
	prompt.WriteString("3. Create meaningful test cases with edge cases and error scenarios\n")
	prompt.WriteString("4. Make sure the code compiles and runs correctly\n")
	prompt.WriteString("5. CRITICAL: Each test MUST actually call the functions to achieve code coverage\n\n")

	prompt.WriteString("SOURCE FILE TO TEST:\n")
	prompt.WriteString("```go\n")
	prompt.WriteString(sourceContent)
	prompt.WriteString("\n```\n\n")

	prompt.WriteString("OUTPUT: Generate ONLY the complete test file content. Start with package declaration and include all necessary imports.\n")

	return prompt.String()
}

// buildEnhancementPrompt - prompt for enhancing existing test file
func (ktg *KubeEdgeTestGenerator) buildEnhancementPrompt(_ string, sourceContent string, existingTestContent string, previousError error) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert Go test generator for KubeEdge project.\n\n")
	
	if previousError != nil {
		prompt.WriteString("PREVIOUS ATTEMPT FAILED:\n")
		prompt.WriteString(fmt.Sprintf("Error: %v\n", previousError))
		prompt.WriteString("Please fix the issues and generate working code.\n\n")
	}

	prompt.WriteString("TASK: Enhance and complete the existing test file to achieve better coverage.\n\n")

	prompt.WriteString("REQUIREMENTS:\n")
	prompt.WriteString("1. Review the existing test file and source code\n")
	prompt.WriteString("2. Add missing tests for any untested exported functions\n")
	prompt.WriteString("3. Improve existing tests with more edge cases and error scenarios\n")
	prompt.WriteString("4. Use whatever imports are required (testing, errors, etc.)\n")
	prompt.WriteString("5. Keep existing good tests and enhance/add new ones\n")
	prompt.WriteString("6. Make sure the code compiles and runs correctly\n")
	prompt.WriteString("7. CRITICAL: Ensure all exported functions are tested\n\n")

	prompt.WriteString("SOURCE FILE:\n")
	prompt.WriteString("```go\n")
	prompt.WriteString(sourceContent)
	prompt.WriteString("\n```\n\n")

	prompt.WriteString("EXISTING TEST FILE:\n")
	prompt.WriteString("```go\n")
	prompt.WriteString(existingTestContent)
	prompt.WriteString("\n```\n\n")

	prompt.WriteString("OUTPUT: Generate ONLY the complete enhanced test file content. Start with package declaration and include all necessary imports.\n")

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


// Close closes the Gemini client
func (ktg *KubeEdgeTestGenerator) Close() {
	if ktg.client != nil {
		ktg.client.Close()
	}
}

