package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// FieldSpec is a resolved struct field — name, type string, and memory dimensions.
type FieldSpec struct {
	Name    string
	TypeStr string
	Size    int
	Align   int
}

// parseStruct accepts a raw Go struct definition in any of these forms:
//
//	type Name struct { ... }
//	struct { ... }
//	{ ... }
func parseStruct(input string) ([]FieldSpec, string, error) {
	input = strings.TrimSpace(input)

	name := "Struct"
	var src string

	switch {
	case strings.HasPrefix(input, "type "):
		src = "package p\n" + input
	case strings.HasPrefix(input, "struct"):
		src = "package p\ntype Struct " + input
		name = "Struct"
	case strings.HasPrefix(input, "{"):
		src = "package p\ntype Struct struct " + input
		name = "Struct"
	default:
		src = "package p\n" + input
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return nil, "", fmt.Errorf("parse error: %w", err)
	}

	var fields []FieldSpec

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			name = ts.Name.Name
			for _, field := range st.Fields.List {
				size, align, err := exprDimensions(field.Type)
				if err != nil {
					return nil, "", fmt.Errorf("field %v: %w", field.Names, err)
				}
				typeStr := exprString(field.Type)
				if len(field.Names) == 0 {
					fields = append(fields, FieldSpec{
						Name:    typeStr,
						TypeStr: typeStr,
						Size:    size,
						Align:   align,
					})
				} else {
					for _, ident := range field.Names {
						fields = append(fields, FieldSpec{
							Name:    ident.Name,
							TypeStr: typeStr,
							Size:    size,
							Align:   align,
						})
					}
				}
			}
		}
	}

	if len(fields) == 0 {
		return nil, "", fmt.Errorf("no fields found — make sure the input is a valid Go struct")
	}

	return fields, name, nil
}

func exprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprString(t.Elt)
		}
		return "[" + exprString(t.Len) + "]" + exprString(t.Elt)
	case *ast.BasicLit:
		return t.Value
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + exprString(t.Key) + "]" + exprString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.ChanType:
		return "chan " + exprString(t.Value)
	case *ast.FuncType:
		return "func(...)"
	case *ast.StructType:
		return "struct{...}"
	default:
		return "?"
	}
}

func exprDimensions(expr ast.Expr) (size, align int, err error) {
	switch t := expr.(type) {
	case *ast.Ident:
		return basicTypeDimensions(t.Name)
	case *ast.StarExpr:
		return 8, 8, nil
	case *ast.ArrayType:
		if t.Len == nil {
			return 24, 8, nil // slice header: ptr + len + cap
		}
		n, e := evalIntLit(t.Len)
		if e != nil {
			return 0, 0, fmt.Errorf("array length: %w", e)
		}
		es, ea, e := exprDimensions(t.Elt)
		if e != nil {
			return 0, 0, e
		}
		// [N]T size = N * sizeof(T), alignment = align(T)
		return n * es, ea, nil
	case *ast.MapType:
		return 8, 8, nil // map is a pointer internally
	case *ast.InterfaceType:
		return 16, 8, nil // type pointer + data pointer
	case *ast.ChanType:
		return 8, 8, nil // channel is a pointer
	case *ast.FuncType:
		return 8, 8, nil // function value is a pointer
	case *ast.SelectorExpr:
		return 0, 0, fmt.Errorf("external type %q not supported — only built-in types are", exprString(expr))
	case *ast.StructType:
		// Inline anonymous struct — compute recursively.
		var sub []FieldSpec
		for _, field := range t.Fields.List {
			fs, fa, fe := exprDimensions(field.Type)
			if fe != nil {
				return 0, 0, fe
			}
			ts := exprString(field.Type)
			if len(field.Names) == 0 {
				sub = append(sub, FieldSpec{Name: ts, TypeStr: ts, Size: fs, Align: fa})
			} else {
				for _, n := range field.Names {
					sub = append(sub, FieldSpec{Name: n.Name, TypeStr: ts, Size: fs, Align: fa})
				}
			}
		}
		layout := computeLayout(sub)
		return layout.StructSize, layout.StructAlign, nil
	default:
		return 0, 0, fmt.Errorf("unsupported type expression: %T", expr)
	}
}

func basicTypeDimensions(name string) (size, align int, err error) {
	table := map[string][2]int{
		"bool":       {1, 1},
		"byte":       {1, 1},
		"uint8":      {1, 1},
		"int8":       {1, 1},
		"int16":      {2, 2},
		"uint16":     {2, 2},
		"int32":      {4, 4},
		"uint32":     {4, 4},
		"float32":    {4, 4},
		"rune":       {4, 4},
		"int64":      {8, 8},
		"uint64":     {8, 8},
		"float64":    {8, 8},
		"complex64":  {8, 8},
		"int":        {8, 8},
		"uint":       {8, 8},
		"uintptr":    {8, 8},
		"string":     {16, 8},
		"complex128": {16, 8},
		"error":      {16, 8},
	}
	if v, ok := table[name]; ok {
		return v[0], v[1], nil
	}
	return 0, 0, fmt.Errorf("unknown type %q (only built-in Go types are supported)", name)
}

func evalIntLit(expr ast.Expr) (int, error) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		return 0, fmt.Errorf("non-literal array length not supported")
	}
	n, err := strconv.Atoi(lit.Value)
	if err != nil {
		return 0, fmt.Errorf("invalid integer literal: %w", err)
	}
	return n, nil
}
