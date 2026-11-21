package utils

import (
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

// AssertStringEqual triggers testing error if expect and actual string are not the same.
func AssertStringEqual(t *testing.T, expect, actual, errMsg string) {
	if expect != actual {
		t.Errorf("%s, expect: \"%s\", actual: \"%s\"", errMsg, expect, actual)
	}
}

func TestIsPathMatch(t *testing.T) {
	cases := []struct {
		name string
		reg  string
		path string
		want bool
	}{
		{"exact", "/default/api", "/default/api", true},
		{"trailing slash in path", "/default/api", "/default/api/", true},
		{"sub path", "/default/api", "/default/api/users/1", true},
		{"segment prefix", "/default/api", "/default/apiv2", false},
		{"segment prefix with dash", "/default/api", "/default/api-internal", false},
		{"segment prefix with sub path", "/default/api", "/default/apikeys/secret", false},
		{"matched at an offset", "/default/api", "/other/default/api", false},
		{"shorter than rule", "/default/api", "/default", false},
		{"empty rule path without a suffix", "/default/", "/default", false},
		{"empty rule path with a trailing slash", "/default/", "/default/", true},
		{"empty rule path with sub path", "/default/", "/default/anything/deeper", true},
		{"literal dot", "/default/api.v1", "/default/api.v1", true},
		{"literal dot is not a wildcard", "/default/api.v1", "/default/apiXv1", false},
		{"literal plus", "/default/api+v1", "/default/api+v1", true},
		{"literal plus is not a quantifier", "/default/api+v1", "/default/apiv1", false},
		{"literal parenthesis", "/default/a(b", "/default/a(b", true},
		{"literal braces", "/default/{}", "/default/{}", true},
		{"param segment", "/default/a/{id}", "/default/a/5", true},
		{"param segment with sub path", "/default/a/{id}", "/default/a/5/sub", true},
		{"param segment missing", "/default/a/{id}", "/default/a", false},
		{"param does not span segments", "/default/a/{id}/end", "/default/a/5/6/end", false},
		{"different namespace", "/default/api", "/other/api", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsPathMatch(c.reg, c.path); got != c.want {
				t.Fatalf("IsPathMatch(%q, %q) = %v, want %v", c.reg, c.path, got, c.want)
			}
		})
	}
}
