package utils

import (
	"k8s.io/klog/v2"
	"regexp"
	"strings"
)

const (
	ParamRegex = "{[-A-Za-z0-9+&@#%?=~_|!:,.;]+}"
	pathRegex  = "[-A-Za-z0-9+&@#%?=~_|!:,.;]+"
)

// URLToURLRegex return url regex and replace {} in url
func URLToURLRegex(url string) string {
	paramRegex := regexp.MustCompile(ParamRegex)
	params := paramRegex.FindAllString(url, -1)
	for _, param := range params {
		url = strings.Replace(url, param, pathRegex, -1)
	}
	url = url + "/{0,1}"
	return url
}

// isMatch return true if the request uri math rule using regex
func IsMatch(reg, path string) bool {
	match, err := regexp.MatchString(URLToURLRegex(reg), path)
	if err != nil {
		klog.Errorf("failed to validate res %s and reqPath %s, err: %v", reg, path, err)
		return false
	}
	if match{
		return true
	}
	return false
}

// RuleContains return true if rule 1 contains rule 2
func RuleContains(rulePath []string, rule2Path []string) bool {
	if len(rulePath) == 0 {
		return true
	}

	if len(rule2Path) == 0 {
		return false
	}

	// rule1[i] contains rule2[i] when rule1[i] = {} or rule1[i] == rule2[i]
	match1, err := regexp.MatchString(URLToURLRegex(rulePath[0]), rule2Path[0])
	if err != nil {
		klog.Errorf("failed to validate rulePath %s and rule2Path %s, err: %v", rulePath[0], rule2Path[0], err)
		return false
	}
	match2, err := regexp.MatchString(ParamRegex, rule2Path[0])
	if err != nil {
		klog.Errorf("failed to validate rule2Path %s, err: %v", rule2Path[0], err)
		return false
	}
	match3, err := regexp.MatchString(ParamRegex, rulePath[0])
	if err != nil {
		klog.Errorf("failed to validate rulePath %s, err: %v", rulePath[0], err)
		return false
	}
	if rulePath[0] == rule2Path[0] || match1 || (match2 && match3) {
		return RuleContains(rulePath[1:], rule2Path[1:])
	}
	return false
}
