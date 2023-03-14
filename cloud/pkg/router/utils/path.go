package utils

import (
	"bytes"
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

// NormalizeResource normalize resource, e.g. path /a/b/ return a/b
func NormalizeResource(resource string) string {
	return strings.Trim(resource, "/")
}

// TrimPrefixByRegex trim prefix reqPath by the rest rule, e.g. path /a/b/c and /a/{a} return /c
func TrimPrefixByRegex(reqPath, rulePath string) string {
	prefix := regexp.MustCompile(URLToURLRegex(rulePath)).FindString(reqPath)
	return strings.TrimPrefix(reqPath, prefix)
}

// MqttToMqttRegex return mqtt regex and replace + and # in topic, e.g. a/+ return a/([^/]*?) and a/# return a[/]?(.*)
func MqttToMqttRegex(topic string) string {
	var buffer bytes.Buffer
	fields := strings.Split(topic, "/")
	for _, field := range fields {
		switch field {
		case "":
			continue
		case "+":
			buffer.WriteString("([^/]*?)")
		case "#":
			buffer.Truncate(buffer.Len() - 1)
			buffer.WriteString("[/]?(.*)")
		default:
			buffer.WriteString(regexp.QuoteMeta(field))
		}
		buffer.WriteString("/")
	}
	topicRegex := "^" + strings.TrimRight(buffer.String(), "/") + "$"
	return topicRegex
}

// IsMqttTopicMatch return true if the topic match mqtt rule using regex
func IsMqttTopicMatch(rule, topic string) bool {
	match, err := regexp.MatchString(MqttToMqttRegex(rule), topic)
	if err != nil {
		klog.ErrorS(err, "fail to match mqtt topic", "rule", rule, "topic", topic)
	}
	return match
}
