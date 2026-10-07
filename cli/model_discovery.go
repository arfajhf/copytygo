package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ModelField struct {
	Name string `json:"name"`
	Type string `json:"type"`
	JSON string `json:"json,omitempty"`
}

type ProjectModel struct {
	Name   string       `json:"name"`
	File   string       `json:"file"`
	Fields []ModelField `json:"fields"`
}

func DiscoverModels(directory string) ([]ProjectModel, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return []ProjectModel{}, nil
		}
		return nil, err
	}

	models := make([]ProjectModel, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		path := filepath.Join(directory, entry.Name())
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}

		for _, declaration := range file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				model := ProjectModel{
					Name: typeSpec.Name.Name,
					File: entry.Name(),
					Fields: make([]ModelField, 0),
				}
				for _, field := range structType.Fields.List {
					if len(field.Names) == 0 {
						continue
					}
					typeName := sourceExpr(field.Type)
					jsonName := ""
					if field.Tag != nil {
						tag := strings.Trim(field.Tag.Value, "`")
						for _, part := range strings.Fields(tag) {
							if strings.HasPrefix(part, "json:") {
								value := strings.TrimPrefix(part, "json:")
								value = strings.Trim(value, "\"")
								jsonName = strings.Split(value, ",")[0]
								break
							}
						}
					}
					for _, name := range field.Names {
						model.Fields = append(model.Fields, ModelField{
							Name: name.Name,
							Type: typeName,
							JSON: jsonName,
						})
					}
				}
				models = append(models, model)
			}
		}
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].Name < models[j].Name
	})
	return models, nil
}

func sourceExpr(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return "*" + sourceExpr(typed.X)
	case *ast.ArrayType:
		return "[]" + sourceExpr(typed.Elt)
	case *ast.MapType:
		return "map[" + sourceExpr(typed.Key) + "]" + sourceExpr(typed.Value)
	case *ast.SelectorExpr:
		return sourceExpr(typed.X) + "." + typed.Sel.Name
	case *ast.InterfaceType:
		return "any"
	default:
		return "unknown"
	}
}
