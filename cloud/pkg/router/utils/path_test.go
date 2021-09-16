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

// AssertStringEqual triggers testing error if the expect and actual string are not the same.
func AssertStringEqual(t *testing.T, expect, actual, errMsg string) {
	if expect != actual {
		t.Errorf("%s, expect: \"%s\", actual: \"%s\"", errMsg, expect, actual)
	}
}
