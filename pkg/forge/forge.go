package forge

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// InterfaceDef represents the extracted blueprint of a Go interface.
type InterfaceDef struct {
	Name          string
	SourcePackage string
	SourcePath    string
	Methods       []MethodDef
}

// MethodDef details a single method signature inside the interface.
type MethodDef struct {
	Name    string
	Params  []FieldDef
	Results []FieldDef
}

// FieldDef represents a parameter or return value.
type FieldDef struct {
	Name string // Can be blank for return types
	Type string
}

// ParseInterface inspects a Go file and extracts a specific interface definition.
func ParseInterface(filepath string, interfaceName string) (*InterfaceDef, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filepath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	var targetInterface *InterfaceDef

	// Walk the AST to look for type declarations
	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != interfaceName {
			return true // Keep looking
		}

		interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
		if !ok {
			return true // Found the name, but it is not an interface
		}

		targetInterface = &InterfaceDef{
			Name: interfaceName,
		}

		// Extract methods
		for _, method := range interfaceType.Methods.List {
			if len(method.Names) == 0 {
				continue // Skip embedded interfaces for simplicity now
			}

			methodName := method.Names[0].Name
			funcType, ok := method.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			mDef := MethodDef{Name: methodName}

			// Parse input parameters
			if funcType.Params != nil {
				mDef.Params = extractFields(funcType.Params.List)
			}

			// Parse return arguments
			if funcType.Results != nil {
				mDef.Results = extractFields(funcType.Results.List)
			}

			targetInterface.Methods = append(targetInterface.Methods, mDef)
		}

		return false // Found our target, stop inspecting
	})

	if targetInterface == nil {
		return nil, fmt.Errorf("interface %s not found in %s", interfaceName, filepath)
	}

	return targetInterface, nil
}

// extractFields converts AST fields into our simple IR definition.
func extractFields(fields []*ast.Field) []FieldDef {
	var result []FieldDef
	for _, field := range fields {
		// Formats types like *User, context.Context, or int into a string
		typeName := exprToString(field.Type)

		if len(field.Names) == 0 {
			// Anonymous field (common in return values)
			result = append(result, FieldDef{Type: typeName})
		} else {
			// Named fields (common in input parameters)
			for _, name := range field.Names {
				result = append(result, FieldDef{Name: name.Name, Type: typeName})
			}
		}
	}
	return result
}

// exprToString converts basic AST expressions into their code string equivalents.
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	default:
		return fmt.Sprintf("%T", expr) // Fallback for complex types
	}
}
