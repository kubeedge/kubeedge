package schema

import (
	"errors"
	"fmt"
	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/pkg/util/fileutil"
	"github.com/go-mesh/openlogging"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MicroserviceMeta is the struct for micro service meta
type MicroserviceMeta struct {
	MicroserviceName string
	SchemaIDs        []string
}

// NewMicroserviceMeta gives the object of MicroserviceMeta
func NewMicroserviceMeta(microserviceName string) *MicroserviceMeta {
	return &MicroserviceMeta{
		MicroserviceName: microserviceName,
		SchemaIDs:        make([]string, 0),
	}
}

// defaultMicroserviceMetaMgr default micro-service meta-manager
var defaultMicroserviceMetaMgr map[string]*MicroserviceMeta

// DefaultSchemaIDsMap default schema schema IDs map
var DefaultSchemaIDsMap map[string]string

// defaultMicroServiceNames default micro-service names
var defaultMicroServiceNames = make([]string, 0)

//GetSchemaPath calculate the schema root path and return
func GetSchemaPath(name string) string {
	schemaEnv := os.Getenv(common.EnvSchemaRoot)
	var p string
	if schemaEnv != "" {
		p = filepath.Join(schemaEnv, name, fileutil.SchemaDirectory)
	} else {
		p = fileutil.SchemaDir(name)
	}
	return p
}

// LoadSchema to load the schema files and micro-service information under the conf directory
func LoadSchema(path string) error {
	/*
		conf/
		├── chassis.yaml
		├── microservice1
		│   └── schema
		│       ├── schema1.yaml
		└── microservice2
		    └── schema
		        ├── schema2.yaml
	*/
	schemaNames, err := getSchemaNames(path)
	if err != nil {
		return err
	}

	for _, msName := range schemaNames {
		var (
			microsvcMeta *MicroserviceMeta
			schemaError  error
		)
		p := GetSchemaPath(msName)
		microsvcMeta, schemaError = loadMicroserviceMeta(GetSchemaPath(msName))

		if schemaError != nil {
			return schemaError
		}

		defaultMicroserviceMetaMgr[msName] = microsvcMeta
		openlogging.Info(fmt.Sprintf("found schema files in %s %s", p, microsvcMeta))
	}
	return nil
}

// getSchemaNames 目录名为服务名
func getSchemaNames(confDir string) ([]string, error) {
	schemaNames := make([]string, 0)
	// 遍历confDir下的microservice文件夹
	err := filepath.Walk(confDir,
		func(path string, info os.FileInfo, err error) error {
			if info == nil {
				return err
			}
			// 仅读取负一级目录
			if !info.IsDir() || filepath.Dir(path) != confDir {
				return nil
			}
			schemaNames = append(schemaNames, info.Name())
			return nil
		})
	return schemaNames, err
}

// SetMicroServiceNames set micro service names
func SetMicroServiceNames(confDir string) error {
	fileFormatName := `microservice(\.yaml|\.yml)$`

	err := filepath.Walk(confDir,
		func(path string, info os.FileInfo, err error) error {
			if info == nil {
				return err
			}
			// 仅读取负一级目录
			if !info.IsDir() || filepath.Dir(path) != confDir {
				return nil
			}

			filesExist, err := getFiles(filepath.Join(confDir, info.Name()))
			if err != nil {
				return err
			}

			for _, name := range filesExist {
				ret, _ := regexp.MatchString(fileFormatName, name)
				if ret {
					defaultMicroServiceNames = append(defaultMicroServiceNames, info.Name())
				}
			}

			return nil
		})
	return err
}

// loadMicroserviceMeta load micro-service meta
func loadMicroserviceMeta(schemaPath string) (*MicroserviceMeta, error) {
	microserviceMeta := NewMicroserviceMeta(filepath.Base(schemaPath))
	schemaFiles, err := getFiles(schemaPath)
	if err != nil {
		return microserviceMeta, err
	}

	for _, fullPath := range schemaFiles {
		schemaFile := filepath.Base(fullPath)
		dat, err := ioutil.ReadFile(fullPath)
		if err != nil {
			e := fmt.Sprintf("The system cannot find the schema file")
			return nil, errors.New(e)
		}

		schemaID := strings.TrimSuffix(schemaFile, filepath.Ext(schemaFile))
		microserviceMeta.SchemaIDs = append(microserviceMeta.SchemaIDs, schemaID)
		DefaultSchemaIDsMap[schemaID] = string(dat)
	}

	return microserviceMeta, nil
}

// getFiles get files
func getFiles(fPath string) ([]string, error) {
	files := make([]string, 0)
	_, err := os.Stat(fPath)
	if os.IsNotExist(err) {
		return files, nil
	}
	// schema文件名规则
	pat := `^.+(\.yaml|\.yml)$`
	// 遍历schemaPath下的schema文件
	err = filepath.Walk(fPath,
		func(path string, info os.FileInfo, err error) error {
			if info == nil {
				return err
			}
			// 仅读取负一级文件
			if info.IsDir() || filepath.Dir(path) != fPath {
				return nil
			}
			ret, _ := regexp.MatchString(pat, info.Name())
			if !ret {
				return nil
			}
			files = append(files, path)
			return nil
		})
	return files, err
}

// GetMicroserviceNamesBySchemas get micro-service names by schemas
func GetMicroserviceNamesBySchemas() []string {
	names := make([]string, 0)
	for k := range defaultMicroserviceMetaMgr {
		names = append(names, k)
	}
	return names
}

// GetMicroserviceNames get micro-service names
func GetMicroserviceNames() []string {
	return defaultMicroServiceNames
}

// GetSchemaIDs get schema IDs
func GetSchemaIDs(microserviceName string) ([]string, error) {
	microsvcMeta, ok := defaultMicroserviceMetaMgr[microserviceName]
	if !ok {
		return nil, fmt.Errorf("microservice %s not found", microserviceName)
	}
	schemaIDs := make([]string, 0)
	for _, v := range microsvcMeta.SchemaIDs {
		schemaIDs = append(schemaIDs, v)
	}
	return schemaIDs, nil
}

// init is for to initialize the defaultMicroserviceMetaMgr, and DefaultSchemaIDsMap
func init() {
	defaultMicroserviceMetaMgr = make(map[string]*MicroserviceMeta)
	DefaultSchemaIDsMap = make(map[string]string)
}
