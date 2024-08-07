package helm

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

// Inspired by https://github.com/istio/istio/blob/194bc3c820a37a38ef40a1cf305529638fbfa169/operator/pkg/helm/renderer.go

const (
	// YAMLSeparator is a separator for multi-document YAML files.
	YAMLSeparator = "\n---\n"

	// DefaultProfileString is the name of the default profile.
	DefaultProfileString = "version"

	// NotesFileNameSuffix is the file name suffix for helm notes.
	// see https://helm.sh/docs/chart_template_guide/notes_files/
	NotesFileNameSuffix = ".txt"

	// Chart Release Name
	ReleaseName = "kubeedge"
)

type TemplateFilterFunc func(string) bool

// Renderer is a helm template renderer for a fs.FS.
type Renderer struct {
	namespace      string
	componentName  string
	chart          *chart.Chart
	files          fs.FS
	dir            string
	profileValsMap map[string]interface{}
	skipCRDs       bool
}

// NewFileTemplateRenderer creates a TemplateRenderer with the given parameters and returns a pointer to it.
// helmChartDirPath must be an absolute file path to the root of the helm charts.
func NewGenericRenderer(files fs.FS, dir, componentName, namespace string, profileValsMap map[string]interface{}, skipCRDs bool) *Renderer {
	return &Renderer{
		namespace:      namespace,
		componentName:  componentName,
		dir:            dir,
		files:          files,
		profileValsMap: profileValsMap,
		skipCRDs:       skipCRDs,
	}
}

// LoadChart would load the given charts.
func (h *Renderer) LoadChart() error {
	return h.loadChart()
}

// RenderManifest renders the current helm templates with the current values and returns the resulting YAML manifest string.
func (h *Renderer) RenderManifest() (string, error) {
	return h.renderChart(nil)
}

// RenderManifestFiltered filters templates to render using the supplied filter function.
func (h *Renderer) RenderManifestFiltered(filter TemplateFilterFunc) (string, error) {
	return h.renderChart(filter)
}

// loadChart implements the TemplateRenderer interface.
func (h *Renderer) loadChart() error {
	fnames, err := GetFilesRecursive(h.files, h.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("component %q does not exist", h.componentName)
		}
		return fmt.Errorf("list files: %v", err)
	}
	var bfs []*loader.BufferedFile
	for _, fname := range fnames {
		b, err := fs.ReadFile(h.files, fname)
		if err != nil {
			return fmt.Errorf("read file: %v", err)
		}
		// Helm expects unix / separator, but on windows this will be \
		name := strings.ReplaceAll(stripPrefix(fname, h.dir), string(filepath.Separator), "/")
		bf := &loader.BufferedFile{
			Name: name,
			Data: b,
		}
		bfs = append(bfs, bf)
	}

	h.chart, err = loader.LoadFiles(bfs)
	if err != nil {
		return fmt.Errorf("load files: %v", err)
	}
	return nil
}

// renderChart renders the given chart with the given values and returns the resulting YAML manifest string.
func (h *Renderer) renderChart(filterFunc TemplateFilterFunc) (string, error) {
	options := chartutil.ReleaseOptions{
		Name:      ReleaseName,
		Namespace: h.namespace,
	}

	caps := *chartutil.DefaultCapabilities
	vals, err := chartutil.ToRenderValues(h.chart, h.profileValsMap, options, &caps)
	if err != nil {
		return "", err
	}

	if filterFunc != nil {
		filteredTemplates := []*chart.File{}
		for _, t := range h.chart.Templates {
			if filterFunc(t.Name) {
				filteredTemplates = append(filteredTemplates, t)
			}
		}
		h.chart.Templates = filteredTemplates
	}

	files, err := engine.Render(h.chart, vals)
	crdFiles := h.chart.CRDObjects()
	if err != nil {
		return "", err
	}

	// Create sorted array of keys to iterate over, to stabilize the order of the rendered templates
	keys := make([]string, 0, len(files))
	for k := range files {
		if strings.HasSuffix(k, NotesFileNameSuffix) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i := 0; i < len(keys); i++ {
		f := files[keys[i]]
		// add yaml separator if the rendered file doesn't have one at the end
		f = strings.TrimSpace(f) + "\n"
		if !strings.HasSuffix(f, YAMLSeparator) {
			f += YAMLSeparator
		}
		_, err := sb.WriteString(f)
		if err != nil {
			return "", err
		}
	}

	// Sort crd files by name to ensure stable manifest output
	if !h.skipCRDs {
		sort.Slice(crdFiles, func(i, j int) bool { return crdFiles[i].Name < crdFiles[j].Name })
		for _, crdFile := range crdFiles {
			f := string(crdFile.File.Data)
			// add yaml separator if the rendered file doesn't have one at the end
			f = strings.TrimSpace(f) + "\n"
			if !strings.HasSuffix(f, YAMLSeparator) {
				f += YAMLSeparator
			}
			_, err := sb.WriteString(f)
			if err != nil {
				return "", err
			}
		}
	}

	return sb.String(), nil
}

func GetFilesRecursive(f fs.FS, root string) ([]string, error) {
	res := []string{}
	err := fs.WalkDir(f, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		res = append(res, path)
		return nil
	})
	return res, err
}

// stripPrefix removes the given prefix from prefix.
func stripPrefix(path, prefix string) string {
	pl := len(strings.Split(prefix, "/"))
	pv := strings.Split(path, "/")
	return strings.Join(pv[pl:], "/")
}
