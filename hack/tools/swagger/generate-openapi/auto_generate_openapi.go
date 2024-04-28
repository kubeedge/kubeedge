package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	rootDir, _ := os.Getwd()
	apisDir := filepath.Join(rootDir, "pkg", "apis")

	var inputDirs []string
	// Traverse the apis directory to find the deepest folders.
	err := filepath.Walk(apisDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != apisDir && !strings.Contains(path, "componentconfig") {
			files, err := os.ReadDir(path)
			if err != nil {
				return err
			}

			// If the directory is empty or does not contain any other subdirectories, then it is the deepest directory.
			hasSubdir := false
			for _, file := range files {
				if file.IsDir() {
					hasSubdir = true
					break
				}
			}

			if !hasSubdir {
				inputDirs = append(inputDirs, path)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %q: %v\n", apisDir, err)
		return
	}
	// Join all the api directories using commas.
	inputDirsParam := strings.Join(inputDirs, ",")
	fmt.Println("The following directories will be included in input-dirs:")
	fmt.Println(inputDirsParam)

	// Execute the openapi-gen command
	fmt.Println("Executing openapi-gen...")
	cmd := exec.Command("go", "run", "vendor/k8s.io/kube-openapi/cmd/openapi-gen/openapi-gen.go",
		"--input-dirs", inputDirsParam,
		"--input-dirs", "k8s.io/apimachinery/pkg/apis/meta/v1",
		"--input-dirs", "k8s.io/api/rbac/v1",
		"--input-dirs", "k8s.io/api/core/v1",
		"--input-dirs", "github.com/kubeedge/kubeedge/pkg/apis",
		"--input-dirs", "k8s.io/apimachinery/pkg/runtime",
		"--input-dirs", "k8s.io/apiextensions-apiserver/pkg/apis",
		"--input-dirs", "k8s.io/kubernetes/pkg/apis",
		"--input-dirs", "k8s.io/apimachinery/pkg/version",
		"--input-dirs", "k8s.io/apimachinery/pkg/api/resource",

		"--output-base", "pkg/generated", // Set the output base directory
		"--output-package", "openapi", // Same as above
		"--go-header-file", "hack/boilerplate/boilerplate.txt", // License file
		"--output-file-base", "zz_generated.openapi", // Set the output name to zz_generated.openapi
		"--v", "9", // Print detailed information
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing openapi-gen: %s\n", err)
	} else {
		fmt.Println("openapi-gen execution completed.")
	}
}
