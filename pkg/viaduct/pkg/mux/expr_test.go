package mux

import (
    "testing"
)

type wantStruct struct {
    pattern    string
    varCount   int
    varNames   []string
    matches    []string
    nonMatches []string
}

func TestMessageExpression(t *testing.T) {
    tests := []struct {
        name     string
        resource string
        want     wantStruct
    }{
        {
            name: "simple_static_path",
            resource: "/devices/light",
            want: wantStruct{
                pattern:    "^/devices/light(/.*)?$",
                varCount:   0,
                varNames:   []string{},
                matches:    []string{
                    "/devices/light",
                    "/devices/light/",
                    "/devices/light/status",
                },
                nonMatches: []string{
                    "/devices",
                    "/device/light",
                    "/devices/lights",
                },
            },
        },
        {
            name: "single_variable_path",
            resource: "/devices/{deviceId}",
            want: wantStruct{
                pattern:    "^/devices/([^/]+?)(/.*)?$",
                varCount:   1,
                varNames:   []string{"deviceId"},
                matches:    []string{
                    "/devices/123",
                    "/devices/123/",
                    "/devices/123/status",
                },
                nonMatches: []string{
                    "/devices",
                    "/device/123",
                    "/devices//",
                },
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            expr := NewExpression().GetExpression(tt.resource)
            if expr == nil {
                t.Fatal("expression creation failed")
            }

            if got := expr.Matcher.String(); got != tt.want.pattern {
                t.Errorf("pattern = %q, want %q", got, tt.want.pattern)
            }

            if expr.VarCount != tt.want.varCount {
                t.Errorf("varCount = %d, want %d", expr.VarCount, tt.want.varCount)
            }

            for _, match := range tt.want.matches {
                if !expr.Matcher.MatchString(match) {
                    t.Errorf("should match %q", match)
                }
            }

            for _, nonMatch := range tt.want.nonMatches {
                if expr.Matcher.MatchString(nonMatch) {
                    t.Errorf("should not match %q", nonMatch)
                }
            }
        })
    }
}