package helm

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart"
)

// TestNewGenericRenderer tests the creation of a new Renderer instance
func TestNewGenericRenderer(t *testing.T) {
	mockFS := fstest.MapFS{}
	testDir := "testdir"
	componentName := "testcomponent"
	namespace := "testnamespace"
	profileValsMap := map[string]interface{}{
		"key": "value",
	}
	skipCRDs := true

	renderer := NewGenericRenderer(
		mockFS,
		testDir,
		componentName,
		namespace,
		profileValsMap,
		skipCRDs,
	)

	assert.NotNil(t, renderer)
	assert.Equal(t, namespace, renderer.namespace)
	assert.Equal(t, componentName, renderer.componentName)
	assert.Equal(t, testDir, renderer.dir)
	assert.Equal(t, mockFS, renderer.files)
	assert.Equal(t, profileValsMap, renderer.profileValsMap)
	assert.Equal(t, skipCRDs, renderer.skipCRDs)
}

// TestLoadChart tests the chart loading functionality
func TestLoadChart(t *testing.T) {
	mockFS := fstest.MapFS{
		"testdir/Chart.yaml": &fstest.MapFile{
			Data: []byte(`apiVersion: v2
name: testchart
version: 1.0.0`),
		},
		"testdir/values.yaml": &fstest.MapFile{
			Data: []byte(`key: value`),
		},
		"testdir/templates/deployment.yaml": &fstest.MapFile{
			Data: []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}`),
		},
	}

	renderer := NewGenericRenderer(
		mockFS,
		"testdir",
		"testcomponent",
		"default",
		map[string]interface{}{},
		false,
	)

	err := renderer.LoadChart()

	assert.NoError(t, err)
	assert.NotNil(t, renderer.chart)
	assert.Equal(t, "testchart", renderer.chart.Metadata.Name)
}

// TestLoadChartError tests error handling in chart loading
func TestLoadChartError(t *testing.T) {
	mockFS := fstest.MapFS{}

	renderer := NewGenericRenderer(
		mockFS,
		"nonexistent",
		"testcomponent",
		"default",
		map[string]interface{}{},
		false,
	)

	err := renderer.LoadChart()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "component \"testcomponent\" does not exist")
}

// TestRenderManifest tests the manifest rendering functionality
func TestRenderManifest(t *testing.T) {
	renderer := &Renderer{
		namespace:     "default",
		componentName: "testcomponent",
		chart: &chart.Chart{
			Metadata: &chart.Metadata{
				Name:    "testchart",
				Version: "1.0.0",
			},
			Templates: []*chart.File{
				{
					Name: "templates/deployment.yaml",
					Data: []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}
  namespace: {{ .Release.Namespace }}`),
				},
			},
		},
		profileValsMap: map[string]interface{}{},
		skipCRDs:       false,
	}

	manifest, err := renderer.RenderManifest()

	assert.NoError(t, err)
	assert.Contains(t, manifest, "apiVersion: apps/v1")
	assert.Contains(t, manifest, "kind: Deployment")
	assert.Contains(t, manifest, "name: kubeedge")
	assert.Contains(t, manifest, "namespace: default")
}

// TestRenderManifestFiltered tests the filtered manifest rendering
func TestRenderManifestFiltered(t *testing.T) {
	renderer := &Renderer{
		namespace:     "default",
		componentName: "testcomponent",
		chart: &chart.Chart{
			Metadata: &chart.Metadata{
				Name:    "testchart",
				Version: "1.0.0",
			},
			Templates: []*chart.File{
				{
					Name: "templates/deployment.yaml",
					Data: []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}`),
				},
				{
					Name: "templates/service.yaml",
					Data: []byte(`apiVersion: v1
kind: Service
metadata:
  name: {{ .Release.Name }}`),
				},
			},
		},
		profileValsMap: map[string]interface{}{},
		skipCRDs:       false,
	}

	filter := func(name string) bool {
		return name == "templates/deployment.yaml"
	}

	manifest, err := renderer.RenderManifestFiltered(filter)

	assert.NoError(t, err)
	assert.Contains(t, manifest, "kind: Deployment")
	assert.NotContains(t, manifest, "kind: Service")
}

// TestGetFilesRecursive tests the recursive file listing functionality
func TestGetFilesRecursive(t *testing.T) {
	mockFS := fstest.MapFS{
		"root/file1.txt":            &fstest.MapFile{Data: []byte("content1")},
		"root/dir/file2.txt":        &fstest.MapFile{Data: []byte("content2")},
		"root/dir/subdir/file3.txt": &fstest.MapFile{Data: []byte("content3")},
	}

	files, err := GetFilesRecursive(mockFS, "root")

	assert.NoError(t, err)
	assert.Len(t, files, 3)
	assert.Contains(t, files, "root/file1.txt")
	assert.Contains(t, files, "root/dir/file2.txt")
	assert.Contains(t, files, "root/dir/subdir/file3.txt")
}
