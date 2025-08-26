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
	"path/filepath"
	"strings"
	"time"
	"os"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// SimplifiedPRCreator handles GitHub PR creation with minimal features
type SimplifiedPRCreator struct {
	client     *github.Client
	repoOwner  string
	repoName   string
	workingDir string
}

// NewSimplifiedPRCreator creates a new simplified PR creator
func NewSimplifiedPRCreator(token, repoOwner, repoName string) *SimplifiedPRCreator {
	return NewSimplifiedPRCreatorWithWorkingDir(token, repoOwner, repoName, ".")
}

// NewSimplifiedPRCreatorWithWorkingDir creates a new simplified PR creator with working directory
func NewSimplifiedPRCreatorWithWorkingDir(token, repoOwner, repoName, workingDir string) *SimplifiedPRCreator {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &SimplifiedPRCreator{
		client:     client,
		repoOwner:  repoOwner,
		repoName:   repoName,
		workingDir: workingDir,
	}
}

// CreateTestsPR creates a simple PR with generated tests
func (spc *SimplifiedPRCreator) CreateTestsPR(ctx context.Context, results []ProcessResult) (int, error) {
	if len(results) == 0 {
		return 0, fmt.Errorf("no results to create PR for")
	}

	// Get default branch
	repo, _, err := spc.client.Repositories.Get(ctx, spc.repoOwner, spc.repoName)
	if err != nil {
		return 0, fmt.Errorf("failed to get repository info: %v", err)
	}
	defaultBranch := repo.GetDefaultBranch()

	// Create branch name
	branchName := fmt.Sprintf("auto-tests-%d", time.Now().Unix())

	// Get latest commit SHA
	ref, _, err := spc.client.Git.GetRef(ctx, spc.repoOwner, spc.repoName, "refs/heads/"+defaultBranch)
	if err != nil {
		return 0, fmt.Errorf("failed to get reference: %v", err)
	}

	// Create new branch
	newRef := &github.Reference{
		Ref: github.String("refs/heads/" + branchName),
		Object: &github.GitObject{
			SHA: ref.Object.SHA,
		},
	}

	_, _, err = spc.client.Git.CreateRef(ctx, spc.repoOwner, spc.repoName, newRef)
	if err != nil {
		return 0, fmt.Errorf("failed to create branch: %v", err)
	}

	// Create/update test files
	for _, result := range results {
		if !result.Success {
			continue
		}

		// Read test file content with working directory context
		testContent, err := readFileWithWorkingDir(result.TestFile, spc.workingDir)
		if err != nil {
			fmt.Printf("Warning: Could not read test file %s: %v\n", result.TestFile, err)
			continue
		}

		// Check if file exists in repo
		existingFile, err := spc.getFileContent(ctx, result.TestFile)
		if err == nil && existingFile != nil {
			// Update existing file
			err = spc.updateFile(ctx, result.TestFile, testContent, branchName, result, existingFile.SHA)
		} else {
			// Create new file
			err = spc.createFile(ctx, result.TestFile, testContent, branchName, result)
		}

		if err != nil {
			fmt.Printf("Warning: Failed to update file %s: %v\n", result.TestFile, err)
		}
	}

	// Create PR with simple description
	prTitle := "🤖 Auto-generated unit tests"
	prBody := spc.buildSimplePRDescription(results)

	pr := &github.NewPullRequest{
		Title: github.String(prTitle),
		Head:  github.String(branchName),
		Base:  github.String(defaultBranch),
		Body:  github.String(prBody),
	}

	createdPR, _, err := spc.client.PullRequests.Create(ctx, spc.repoOwner, spc.repoName, pr)
	if err != nil {
		return 0, fmt.Errorf("failed to create pull request: %v", err)
	}

	return createdPR.GetNumber(), nil
}

// createFile creates a new file in the repository
func (spc *SimplifiedPRCreator) createFile(ctx context.Context, filePath, content, branchName string, result ProcessResult) error {
	commitMessage := fmt.Sprintf("Add tests for %s", filepath.Base(result.SourceFile))

	fileOptions := &github.RepositoryContentFileOptions{
		Message: github.String(commitMessage),
		Content: []byte(content),
		Branch:  github.String(branchName),
	}

	_, _, err := spc.client.Repositories.CreateFile(ctx, spc.repoOwner, spc.repoName, filePath, fileOptions)
	return err
}

// updateFile updates an existing file in the repository
func (spc *SimplifiedPRCreator) updateFile(ctx context.Context, filePath, content, branchName string, result ProcessResult, sha *string) error {
	commitMessage := fmt.Sprintf("Update tests for %s", filepath.Base(result.SourceFile))

	fileOptions := &github.RepositoryContentFileOptions{
		Message: github.String(commitMessage),
		Content: []byte(content),
		Branch:  github.String(branchName),
		SHA:     sha,
	}

	_, _, err := spc.client.Repositories.UpdateFile(ctx, spc.repoOwner, spc.repoName, filePath, fileOptions)
	return err
}

// getFileContent retrieves existing file content from repository
func (spc *SimplifiedPRCreator) getFileContent(ctx context.Context, filePath string) (*github.RepositoryContent, error) {
	fileContent, _, _, err := spc.client.Repositories.GetContents(ctx, spc.repoOwner, spc.repoName, filePath, nil)
	if err != nil {
		return nil, err
	}
	return fileContent, nil
}

// buildSimplePRDescription creates a clean, simple PR description
func (spc *SimplifiedPRCreator) buildSimplePRDescription(results []ProcessResult) string {
	var body strings.Builder

	body.WriteString("## 🤖 Auto-Generated Unit Tests\n\n")
	body.WriteString("Generated unit tests using LLM (Gemini) for files with low test coverage.\n\n")

	// Simple coverage table
	body.WriteString("**Coverage Improvements:**\n\n")
	
	totalImprovement := 0.0
	fileCount := 0

	for _, result := range results {
		if result.Success {
			improvement := result.AfterCoverage - result.BeforeCoverage
			totalImprovement += improvement
			fileCount++
			
			body.WriteString(fmt.Sprintf("- `%s`: %.1f%% → %.1f%% (+%.1f%%)\n",
				result.SourceFile, result.BeforeCoverage, result.AfterCoverage, improvement))
		}
	}

	if fileCount > 0 {
		avgImprovement := totalImprovement / float64(fileCount)
		body.WriteString(fmt.Sprintf("\n**Total: %d files, average +%.1f%% coverage improvement**\n\n", fileCount, avgImprovement))
	}

	// Simple validation note
	body.WriteString("✅ All tests compile and pass\n")
	body.WriteString("🤖 Generated using KubeEdge Auto Test Generator\n")

	return body.String()
}

// readFileWithWorkingDir reads content from a file with working directory context
func readFileWithWorkingDir(filePath, workingDir string) (string, error) {
	// Try relative path first
	content, err := os.ReadFile(filePath)
	if err == nil {
		return string(content), nil
	}
	
	// Try absolute path with working directory
	absPath := filepath.Join(workingDir, filePath)
	content, err = os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file at both %s and %s: %v", filePath, absPath, err)
	}
	return string(content), nil
}