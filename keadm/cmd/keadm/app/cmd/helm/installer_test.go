package helm

import (
    "testing"
)

func TestHandleProfileWithImageRepository(t *testing.T) {
    tests := []struct {
        name            string
        imageRepository string
        expectedSets    []string
    }{
        {
            name:            "with local registry",
            imageRepository: "localhost:5000",
            expectedSets: []string{
                "cloudCore.image.repository=localhost:5000/cloudcore",
                "iptablesManager.image.repository=localhost:5000/iptables-manager",
                "controllerManager.image.repository=localhost:5000/controller-manager",
            },
        },
        {
            name:            "with empty repository",
            imageRepository: "",
            expectedSets:    []string{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cu := &KubeCloudHelmInstTool{
                ImageRepository: tt.imageRepository,
            }

            err := cu.handleProfile("")
            if err != nil {
                t.Errorf("handleProfile() error = %v", err)
            }

            if len(cu.Sets) != len(tt.expectedSets) {
                t.Errorf("got %d sets, want %d sets", len(cu.Sets), len(tt.expectedSets))
            }

            for _, expected := range tt.expectedSets {
                found := false
                for _, actual := range cu.Sets {
                    if actual == expected {
                        found = true
                        break
                    }
                }
                if !found {
                    t.Errorf("expected set %q not found in %v", expected, cu.Sets)
                }
            }
        })
    }
}