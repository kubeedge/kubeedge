package utils

import (
	"fmt"
	"testing"
)

func TestURLToURLRegex(t *testing.T) {
	uri := "/aaa/bbb/ccc/ddd"
	uriReg := URLToURLRegex(uri)
	AssertStringEqual(t, uri+tailRegex, uriReg, "1")

	uri = "/aaa/bbb/{sssss}/ddd"
	uriReg = URLToURLRegex(uri)
	AssertStringEqual(t, "/aaa/bbb/"+pathRegex+"/ddd"+tailRegex, uriReg, "2")

	uri = "/aaa/{ddddd}/{sssss}/ddd"
	uriReg = URLToURLRegex(uri)
	AssertStringEqual(t, "/aaa/"+pathRegex+"/"+pathRegex+"/ddd"+tailRegex, uriReg, "3")
}

func TestPathMatch(t *testing.T) {
	rule := "/"
	req := "/fakenodeid/a/b/c"
	AssertTrue(t, IsMatch(rule, req), "1")

	rule = "/a/{sdsd}"
	AssertTrue(t, IsMatch(rule, req), "2")

	rule = "/a/{sdsd}/{dddd}"
	AssertTrue(t, IsMatch(rule, req), "3")

	rule = "/a/"
	AssertTrue(t, IsMatch(rule, req), "4")

	rule = "/a"
	AssertTrue(t, IsMatch(rule, req), "5")

	rule = "/a/b/c"
	AssertTrue(t, IsMatch(rule, req), "6")

	rule = "/a/b/d"
	AssertTrue(t, !IsMatch(rule, req), "7")
}

func TestRuleContains(t *testing.T) {
	cases := []struct {
		name      string
		rulePath  string
		rule2Path string
		want      bool
	}{
		{
			name:      "case1",
			rulePath:  "/a",
			rule2Path: "/a/b",
			want:      true,
		},
		{
			name:      "case2",
			rulePath:  "/a",
			rule2Path: "/b",
			want:      false,
		},
	}

	for _, c := range cases {
		if actual := RuleContains(c.rulePath, c.rule2Path); c.want != actual {
			t.Errorf("%v: expected %v, but got %v", c.name, c.want, actual)
		}
	}
}

// AssertTrue triggers testing error if the passed-in is true.
func AssertTrue(t *testing.T, value bool, errMsg string) {
	if !value {
		t.Errorf("error: %s", errMsg)
	}
}

// AssertStringEqual triggers testing error if the expect and actual string are not the same.
func AssertStringEqual(t *testing.T, expect, actual, errMsg string) {
	if expect != actual {
		t.Errorf("%s, expect: \"%s\", actual: \"%s\"", errMsg, expect, actual)
	}
}

func TestNormalizeResource(t *testing.T) {
	tests := []struct {
		resource string
		want     string
	}{
		{
			resource: "/a/b/",
			want:     "a/b",
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if got := NormalizeResource(tt.resource); got != tt.want {
				t.Errorf("NormalizeResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrimPrefixByRegex(t *testing.T) {
	tests := []struct {
		reqPath  string
		rulePath string
		want     string
	}{
		{
			reqPath:  "test/abc/123",
			rulePath: "test/abc/123",
			want:     "",
		},
		{
			reqPath:  "test/abc/123/456",
			rulePath: "test/abc/123",
			want:     "456",
		},
		{
			reqPath:  "test/abc/123",
			rulePath: "test/{a}/123",
			want:     "",
		},
		{
			reqPath:  "test/abc/123/456",
			rulePath: "test/{a}/123",
			want:     "456",
		},
		{
			reqPath:  "test/abc/123/456/789",
			rulePath: "test/{a}/123",
			want:     "456/789",
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if got := TrimPrefixByRegex(tt.reqPath, tt.rulePath); got != tt.want {
				t.Errorf("TrimPrefixByRegex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsMqttTopicMatch(t *testing.T) {
	testCases := []struct {
		rule  string
		topic string
		want  bool
	}{
		{
			rule:  "a",
			topic: "a",
			want:  true,
		},
		{
			rule:  "a/b/c/d",
			topic: "a/b/c/d",
			want:  true,
		},
		{
			rule:  "a",
			topic: "a/b",
			want:  false,
		},
		{
			rule:  "a/b",
			topic: "a",
			want:  false,
		},
		{
			rule:  "a/+",
			topic: "a/b",
			want:  true,
		},
		{
			rule:  "a/+",
			topic: "a/b/c",
			want:  false,
		},
		{
			rule:  "a/#",
			topic: "a/b",
			want:  true,
		},
		{
			rule:  "a/#",
			topic: "a/b/c",
			want:  true,
		},
		{
			rule:  "a/+/+",
			topic: "a/b",
			want:  false,
		},
		{
			rule:  "a/+/+",
			topic: "a/bb/cc",
			want:  true,
		},
		{
			rule:  "a/b/#",
			topic: "a/b",
			want:  true,
		},
		{
			rule:  "a/b/#",
			topic: "a/b/",
			want:  true,
		},
		{
			rule:  "a/b/#",
			topic: "a/b/c",
			want:  true,
		},
		{
			rule:  "a/b/+",
			topic: "a/b",
			want:  false,
		},
		{
			rule:  "a/b/+",
			topic: "a/b/",
			want:  true,
		},
		{
			rule:  "a/b/+",
			topic: "a/b/c",
			want:  true,
		},
		{
			rule:  "a/#",
			topic: "a/+/c",
			want:  true,
		},
		{
			rule:  "a/+/+/c",
			topic: "a/+/b/c",
			want:  true,
		},
		{
			rule:  "a/+/b/c",
			topic: "a/+/+/c",
			want:  false,
		},
		{
			rule:  "a/+/c",
			topic: "a/#",
			want:  false,
		},
		{
			rule:  "a/$b/c",
			topic: "a/$b/c",
			want:  true,
		},
	}
	for i, tt := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if got := IsMqttTopicMatch(tt.rule, tt.topic); got != tt.want {
				t.Errorf("IsMqttTopicMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}
