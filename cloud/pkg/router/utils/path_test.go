package utils

import (
	"strings"
	"testing"
)

func TestURLToURLRegex(t *testing.T) {
	uri := "/aaa/bbb/ccc/ddd"
	uriReg := URLToURLRegex(uri)
	AssertStringEqual(t, uri+"/{0,1}", uriReg, "1")

	uri = "/aaa/bbb/{sssss}/ddd"
	uriReg = URLToURLRegex(uri)
	AssertStringEqual(t, "/aaa/bbb/[-A-Za-z0-9+&@#%?=~_|!:,.;]+/ddd/{0,1}", uriReg, "2")

	uri = "/aaa/{ddddd}/{sssss}/ddd"
	uriReg = URLToURLRegex(uri)
	AssertStringEqual(t, "/aaa/[-A-Za-z0-9+&@#%?=~_|!:,.;]+/[-A-Za-z0-9+&@#%?=~_|!:,.;]+/ddd/{0,1}", uriReg, "3")
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

func normalizeResource(resource string) string {
	finalResource := resource
	if strings.HasPrefix(finalResource, "/") {
		finalResource = "/" + strings.TrimLeft(finalResource, "/")
	}
	if strings.HasSuffix(finalResource, "/") {
		finalResource = strings.TrimRight(finalResource, "/")
	}
	return finalResource
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
