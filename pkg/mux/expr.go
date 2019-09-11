package mux

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/klog"
)

type MessageExpression struct {
	VarNames []string
	VarCount int
	Matcher  *regexp.Regexp
}

func NewExpression() *MessageExpression {
	return &MessageExpression{}
}

func (exp *MessageExpression) GetExpression(resource string) *MessageExpression {
	var buffer bytes.Buffer
	var varNames []string
	var varCount int

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
			if colon != -1 {
				varName = strings.TrimSpace(field[1:colon])
				paramExpr := strings.TrimSpace(field[colon+1 : len(field)-1])
				if paramExpr == "*" {
					buffer.WriteString("(.*)")
				} else {
					buffer.WriteString(fmt.Sprintf("(%s)", paramExpr))
				}
			} else {
				varName = strings.TrimSpace(field[1 : len(field)-1])
				buffer.WriteString("([^/]+?)")
			}
			varNames = append(varNames, varName)
			varCount += 1
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
		Matcher:  compiled,
		VarCount: varCount,
		VarNames: varNames,
	}
}
