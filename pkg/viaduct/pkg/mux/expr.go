package mux

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/klog/v2"
)

type MessageExpression struct {
	VarNames   []string
	VarIndexes []int
	VarCount   int
	Matcher    *regexp.Regexp
}

func NewExpression() *MessageExpression {
	return &MessageExpression{}
}

func (exp *MessageExpression) GetExpression(resource string) *MessageExpression {
	var buffer bytes.Buffer
	var varNames []string
	var varIndexes []int
	var varCount int
	captureIndex := 1

	if resource == "*" {
		compiled, _ := regexp.Compile("(/.*)?$")
		return &MessageExpression{
			Matcher: compiled,
		}
	}

	buffer.WriteString("^")
	if strings.HasPrefix(resource, "/") {
		buffer.WriteString("/")
	}

	fields := strings.Split(strings.Trim(resource, "/"), "/")
	for _, field := range fields {
		if field == "" {
			continue
		}
		if strings.HasPrefix(field, "{") {
			colon := strings.Index(field, ":")
			var varName string
			varIndex := captureIndex
			if colon != -1 {
				varName = strings.TrimSpace(field[1:colon])
				paramExpr := strings.TrimSpace(field[colon+1 : len(field)-1])
				if paramExpr == "*" {
					buffer.WriteString("(.*)")
					captureIndex++
				} else {
					buffer.WriteString(fmt.Sprintf("(%s)", paramExpr))
					captureIndex += 1 + countCapturingGroups(paramExpr)
				}
			} else {
				varName = strings.TrimSpace(field[1 : len(field)-1])
				buffer.WriteString("([^/]+?)")
				captureIndex++
			}
			varNames = append(varNames, varName)
			varIndexes = append(varIndexes, varIndex)
			varCount++
		} else {
			buffer.WriteString(regexp.QuoteMeta(field))
		}
		buffer.WriteString("/")
	}

	expression := strings.TrimRight(buffer.String(), "/") + "(/.*)?$"
	compiled, err := regexp.Compile(expression)
	if err != nil {
		klog.Errorf("failed to compile resource expression(%s)", expression)
		return nil
	}

	return &MessageExpression{
		Matcher:    compiled,
		VarCount:   varCount,
		VarNames:   varNames,
		VarIndexes: varIndexes,
	}
}

func countCapturingGroups(expression string) int {
	var count int
	var escaped bool
	var inCharClass bool
	for index := 0; index < len(expression); index++ {
		switch {
		case escaped:
			escaped = false
		case expression[index] == '\\':
			escaped = true
		case expression[index] == '[':
			inCharClass = true
		case expression[index] == ']':
			inCharClass = false
		case expression[index] == '(' && !inCharClass:
			if isNonCapturingGroup(expression, index) {
				continue
			}
			count++
		}
	}
	return count
}

func isNonCapturingGroup(expression string, index int) bool {
	if index+2 >= len(expression) || expression[index+1] != '?' {
		return false
	}

	switch expression[index+2] {
	case ':', 'i', 'm', 's', 'U', '-':
		return true
	default:
		return false
	}
}
