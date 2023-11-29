package helm

import "testing"

func TestMergeProfileValues(t *testing.T) {
	cases := []struct {
		name string
		file string
	}{
		{
			name: "load cloudcore values.yaml",
			file: "charts/cloudcore/values.yaml",
		},
		{
			name: "load profile values",
			file: "profiles/version.yaml",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vals, err := MergeProfileValues(c.file, []string{})
			if err != nil {
				t.Fatalf("faield to load helm values, err: %v", err)
			}
			if len(vals) == 0 {
				t.Fatal("the value returned is empty")
			}
		})
	}
}
