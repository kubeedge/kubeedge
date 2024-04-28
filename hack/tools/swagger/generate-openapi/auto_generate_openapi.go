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
	//遍历apis目录，寻找深度最深的文件夹
	err := filepath.Walk(apisDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != apisDir && !strings.Contains(path, "componentconfig") {
			files, err := os.ReadDir(path)
			if err != nil {
				return err
			}

			// 如果目录为空，或者目录中没有其他子目录，那么它就是最深层的目录
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
	//使用逗号连接所有的api目录
	inputDirsParam := strings.Join(inputDirs, ",")
	fmt.Println("The following directories will be included in input-dirs:")
	fmt.Println(inputDirsParam)

	// 执行 openapi-gen 命令
	fmt.Println("执行 openapi-gen...")
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

		"--output-base", "pkg/generated", //输出结果设置所在包
		"--output-package", "openapi", //同上
		"--go-header-file", "hack/boilerplate/boilerplate.txt", //许可证文件
		"--output-file-base", "zz_generated.openapi", //设置输出名字为zz_generated.openapi
		"--v", "9", //打印详细信息
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("执行 openapi-gen 出错: %s\n", err)
	} else {
		fmt.Println("openapi-gen 执行完成。")
	}
}
