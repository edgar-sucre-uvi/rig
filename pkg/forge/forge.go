package forge

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

type InterfaceDef struct {
	Name          string
	SourcePackage string // e.g., "domain"
	SourcePath    string // Absolute path to source directory
	Methods       []MethodDef
}

type MethodDef struct {
	Name    string
	Params  []FieldDef
	Results []FieldDef
}

type FieldDef struct {
	Name string
	Type string
}

func ParseInterface(srcFile string, interfaceName string) (*InterfaceDef, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, srcFile, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	absSrcPath, _ := filepath.Abs(srcFile)
	sourcePackage := node.Name.Name

	targetInterface := &InterfaceDef{
		Name:          interfaceName,
		SourcePackage: sourcePackage,
		SourcePath:    filepath.Dir(absSrcPath),
	}

	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != interfaceName {
			return true
		}

		interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
		if !ok {
			return true
		}

		found = true

		for _, method := range interfaceType.Methods.List {
			if len(method.Names) == 0 {
				continue
			}

			methodName := method.Names[0].Name
			funcType, ok := method.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			mDef := MethodDef{Name: methodName}

			if funcType.Params != nil {
				mDef.Params = extractFields(funcType.Params.List)
			}
			if funcType.Results != nil {
				mDef.Results = extractFields(funcType.Results.List)
			}

			targetInterface.Methods = append(targetInterface.Methods, mDef)
		}
		return false
	})

	if !found {
		return nil, fmt.Errorf("interface %s not found in %s", interfaceName, srcFile)
	}

	return targetInterface, nil
}

func extractFields(fields []*ast.Field) []FieldDef {
	var result []FieldDef
	for _, field := range fields {
		typeName := exprToString(field.Type)

		if len(field.Names) == 0 {
			result = append(result, FieldDef{Type: typeName})
		} else {
			for _, name := range field.Names {
				result = append(result, FieldDef{Name: name.Name, Type: typeName})
			}
		}
	}
	return result
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// QualifyTypes dynamically runs through the interface fields and prefixes local
// custom types with the source package name if we are generating code outside the source directory.
func (i *InterfaceDef) QualifyTypes() {
	primitives := map[string]bool{
		"int": true, "int64": true, "string": true, "bool": true, "error": true,
		"float64": true, "context.Context": true, "any": true,
	}

	qualify := func(t string) string {
		hasPointer := strings.HasPrefix(t, "*")
		pureType := strings.TrimPrefix(t, "*")

		// If it's a known primitive or already has a package selector (like context.Context), skip it
		if primitives[pureType] || strings.Contains(pureType, ".") {
			return t
		}

		// Otherwise, forge the correct package path qualifier
		if hasPointer {
			return fmt.Sprintf("*%s.%s", i.SourcePackage, pureType)
		}
		return fmt.Sprintf("%s.%s", i.SourcePackage, pureType)
	}

	for mIdx, method := range i.Methods {
		for pIdx, param := range method.Params {
			i.Methods[mIdx].Params[pIdx].Type = qualify(param.Type)
		}
		for rIdx, res := range method.Results {
			i.Methods[mIdx].Results[rIdx].Type = qualify(res.Type)
		}
	}
}
