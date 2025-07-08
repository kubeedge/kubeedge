package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// FunctionInfo represents information about a function
type FunctionInfo struct {
	Name        string
	IsExported  bool
	Content     string
	Signature   string
	Parameters  []string
	ReturnTypes []string
	StartLine   int
	EndLine     int
	HasTests    bool
	Complexity  int
}

// CoverageAnalyzer provides function analysis capabilities (simplified)
type CoverageAnalyzer struct{}

// NewCoverageAnalyzer creates a new coverage analyzer
func NewCoverageAnalyzer() *CoverageAnalyzer {
	return &CoverageAnalyzer{}
}

// ExtractModifiedFunctions extracts functions from a modified file
func (ca *CoverageAnalyzer) ExtractModifiedFunctions(filePath string) ([]FunctionInfo, error) {
	// Check if file exists
	if !fileExists(filePath) {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}
	
	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Read the file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Extract functions from content
	return ca.extractFunctionsFromContent(string(content), absPath)
}

// extractFunctionsFromContent extracts functions from Go source code content
func (ca *CoverageAnalyzer) extractFunctionsFromContent(content, filePath string) ([]FunctionInfo, error) {
	// Parse the Go source code
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go source: %v", err)
	}

	var functions []FunctionInfo
	lines := strings.Split(content, "\n")

	// Walk the AST to find function declarations
	ast.Inspect(node, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Name != nil && ca.shouldIncludeFunction(fn) {
				funcInfo := ca.extractFunctionInfo(fn, lines, fset)
				funcInfo.Complexity = ca.calculateComplexity(fn)
				functions = append(functions, funcInfo)
			}
		}
		return true
	})

	return functions, nil
}

// shouldIncludeFunction determines if a function should be included for testing
func (ca *CoverageAnalyzer) shouldIncludeFunction(fn *ast.FuncDecl) bool {
	if fn.Name == nil {
		return false
	}

	funcName := fn.Name.Name

	// Skip init functions
	if funcName == "init" {
		return false
	}

	// Skip main function
	if funcName == "main" {
		return false
	}

	// Skip test functions
	if strings.HasPrefix(funcName, "Test") || 
	   strings.HasPrefix(funcName, "Benchmark") || 
	   strings.HasPrefix(funcName, "Example") {
		return false
	}

	// Skip functions with build tags or special comments
	if fn.Doc != nil {
		for _, comment := range fn.Doc.List {
			if strings.Contains(comment.Text, "// +build") ||
			   strings.Contains(comment.Text, "//go:build") ||
			   strings.Contains(comment.Text, "// TODO") ||
			   strings.Contains(comment.Text, "// FIXME") ||
			   strings.Contains(comment.Text, "// Deprecated") {
				return false
			}
		}
	}

	return true
}

// extractFunctionInfo extracts detailed information about a function
func (ca *CoverageAnalyzer) extractFunctionInfo(fn *ast.FuncDecl, lines []string, fset *token.FileSet) FunctionInfo {
	funcName := fn.Name.Name
	
	// Determine if function is exported
	isExported := ast.IsExported(funcName)
	
	// Extract function content
	startPos := fset.Position(fn.Pos())
	endPos := fset.Position(fn.End())
	
	var content strings.Builder
	for i := startPos.Line - 1; i < endPos.Line && i < len(lines); i++ {
		content.WriteString(lines[i])
		if i < endPos.Line-1 {
			content.WriteString("\n")
		}
	}

	// Extract function signature
	signature := ca.extractFunctionSignature(fn)

	// Extract parameters
	var parameters []string
	if fn.Type.Params != nil {
		for _, param := range fn.Type.Params.List {
			for _, name := range param.Names {
				parameters = append(parameters, name.Name)
			}
		}
	}

	// Extract return types
	var returnTypes []string
	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			returnTypes = append(returnTypes, ca.typeToString(result.Type))
		}
	}

	return FunctionInfo{
		Name:        funcName,
		IsExported:  isExported,
		Content:     content.String(),
		Signature:   signature,
		Parameters:  parameters,
		ReturnTypes: returnTypes,
		StartLine:   startPos.Line,
		EndLine:     endPos.Line,
		HasTests:    false, // Will be determined by validator if needed
	}
}

// extractFunctionSignature creates a readable function signature
func (ca *CoverageAnalyzer) extractFunctionSignature(fn *ast.FuncDecl) string {
	var signature strings.Builder
	
	signature.WriteString("func ")
	signature.WriteString(fn.Name.Name)
	signature.WriteString("(")
	
	// Add parameters
	if fn.Type.Params != nil {
		for i, param := range fn.Type.Params.List {
			if i > 0 {
				signature.WriteString(", ")
			}
			for j, name := range param.Names {
				if j > 0 {
					signature.WriteString(", ")
				}
				signature.WriteString(name.Name)
			}
			signature.WriteString(" ")
			signature.WriteString(ca.typeToString(param.Type))
		}
	}
	
	signature.WriteString(")")
	
	// Add return types
	if fn.Type.Results != nil {
		if len(fn.Type.Results.List) == 1 {
			signature.WriteString(" ")
			signature.WriteString(ca.typeToString(fn.Type.Results.List[0].Type))
		} else if len(fn.Type.Results.List) > 1 {
			signature.WriteString(" (")
			for i, result := range fn.Type.Results.List {
				if i > 0 {
					signature.WriteString(", ")
				}
				signature.WriteString(ca.typeToString(result.Type))
			}
			signature.WriteString(")")
		}
	}
	
	return signature.String()
}

// typeToString converts an AST type to string representation
func (ca *CoverageAnalyzer) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return ca.typeToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + ca.typeToString(t.X)
	case *ast.ArrayType:
		return "[]" + ca.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + ca.typeToString(t.Key) + "]" + ca.typeToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.ChanType:
		return "chan " + ca.typeToString(t.Value)
	case *ast.FuncType:
		return "func(...)"
	default:
		return "unknown"
	}
}

// calculateComplexity calculates a simple complexity score for a function
func (ca *CoverageAnalyzer) calculateComplexity(fn *ast.FuncDecl) int {
	complexity := 1 // Base complexity
	
	ast.Inspect(fn, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.SwitchStmt:
			complexity++
		case *ast.TypeSwitchStmt:
			complexity++
		case *ast.SelectStmt:
			complexity++
		}
		return true
	})
	
	return complexity
}