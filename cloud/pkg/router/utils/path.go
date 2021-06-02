package utils

import (
	"regexp"
	"strings"

	"k8s.io/klog/v2"
)

const (
	pathRegex = "[-A-Za-z0-9+&@#%?=~_|!:,.;]+"
	tailRegex = "/?"
)

var paramRegex = regexp.MustCompile("{" + pathRegex + "}")

// URLToURLRegex return url regex and replace {} in url,e.g.,/abc/{Aa1} to /abc/[-A-Za-z0-9+&@#%?=~_|!:,.;]+/?
func URLToURLRegex(url string) string {
	params := paramRegex.FindAllString(url, -1)
	for _, param := range params {
		url = strings.Replace(url, param, pathRegex, -1)
	}
	url = url + tailRegex
	return url
}

// IsMatch return true if the path match rule using regex
func IsMatch(reg, path string) bool {
	match, err := regexp.MatchString(URLToURLRegex(reg), path)
	if err != nil {
		klog.Errorf("failed to validate res %s and reqPath %s, err: %v", reg, path, err)
		return false
	}
	return match
}

// RuleContains return true if rule 1 contains rule 2, e.g., path /a contains /a/b
func RuleContains(rulePath, rule2Path string) bool {
	path1 := strings.Split(rulePath, "/")
	path2 := strings.Split(rule2Path, "/")
	if len(path1) == 0 {
		return true
	}

	if len(path2) == 0 {
		return false
	}

	for i := 0; i < len(path1) && i < len(path2); i++ {
		// rule1[i] contains rule2[i] when rule1[i] = {} or rule1[i] == rule2[i]
		if path1[i] != path2[i] && URLToURLRegex(path1[i]) != URLToURLRegex(path2[i]) {
			return false
		}
	}
	return true
}
