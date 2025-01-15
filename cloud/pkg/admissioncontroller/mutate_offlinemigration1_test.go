package yourpackage

import (
    "testing"
    corev1 "k8s.io/api/core/v1"
    "github.com/stretchr/testify/assert"
)

func TestGeneratePatch(t *testing.T) {
    // Define test cases
    tests := []struct {
        name         string
        tolerations  []corev1.Toleration
        expectedPatch []patchMapValue
    }{
        {
            name: "No tolerations",
            tolerations: []corev1.Toleration{},
            expectedPatch: []patchMapValue{},
        },
        {
            name: "Single toleration with TaintNodeUnreachable",
            tolerations: []corev1.Toleration{
                {
                    Key: corev1.TaintNodeUnreachable,
                    Operator: corev1.TolerationOpExists,
                },
            },
            expectedPatch: []patchMapValue{},
        },
        {
            name: "Single toleration without TaintNodeUnreachable",
            tolerations: []corev1.Toleration{
                {
                    Key: "example.com/key",
                    Operator: corev1.TolerationOpEqual,
                    Value: "value",
                },
            },
            expectedPatch: []patchMapValue{
                {
                    Op:    "add",
                    Path:  "/spec/tolerations/-",
                    Value: corev1.Toleration{
                        Key:      "example.com/key",
                        Operator: corev1.TolerationOpEqual,
                        Value:    "value",
                    },
                },
            },
        },
        {
            name: "Multiple tolerations with one TaintNodeUnreachable",
            tolerations: []corev1.Toleration{
                {
                    Key: corev1.TaintNodeUnreachable,
                    Operator: corev1.TolerationOpExists,
                },
                {
                    Key: "example.com/key1",
                    Operator: corev1.TolerationOpEqual,
                    Value: "value1",
                },
                {
                    Key: "example.com/key2",
                    Operator: corev1.TolerationOpEqual,
                    Value: "value2",
                },
            },
            expectedPatch: []patchMapValue{
                {
                    Op:    "add",
                    Path:  "/spec/tolerations/-",
                    Value: corev1.Toleration{
                        Key:      "example.com/key1",
                        Operator: corev1.TolerationOpEqual,
                        Value:    "value1",
                    },
                },
                {
                    Op:    "add",
                    Path:  "/spec/tolerations/-",
                    Value: corev1.Toleration{
                        Key:      "example.com/key2",
                        Operator: corev1.TolerationOpEqual,
                        Value:    "value2",
                    },
                },
            },
        },
    }

    // Run test cases
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := generatePatch(tt.tolerations)
            assert.Equal(t, tt.expectedPatch, result)
        })
    }
}