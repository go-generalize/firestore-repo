package main

import (
	"fmt"
	"go/ast"

	"golang.org/x/xerrors"
)

type Field struct {
	Name       string
	Type       string
	ParentPath string
	IsEmbed    bool
	IsPointer  bool
	Space      string
}

type MetaField struct {
	Require          bool
	RequireType      string
	Find             bool
	FindType         string
	RequireIsPointer bool
	FindIsPointer    bool
}

func listAllField(field *ast.FieldList, parentName string, isEmbed bool) []*Field {
	result := make([]*Field, 0)

	for _, f := range field.List {
		name := ""
		typeName := ""
		isCurrentEmbed := false
		isPointer := false

		switch f.Type.(type) {
		case *ast.Ident:
			typeName = f.Type.(*ast.Ident).Name
		case *ast.SelectorExpr:
			t := f.Type.(*ast.SelectorExpr)
			if x, ok := t.X.(*ast.Ident); ok {
				typeName = fmt.Sprintf("%s.%s",
					x.Name, t.Sel.Name)
			} else {
				typeName = t.Sel.Name
			}
		case *ast.StarExpr:
			t := f.Type.(*ast.StarExpr)
			if xSel, ok := t.X.(*ast.SelectorExpr); ok {
				isPointer = true
				if x, ok := xSel.X.(*ast.Ident); ok {
					typeName = fmt.Sprintf("%s.%s",
						x.Name, xSel.Sel.Name)
				} else {
					typeName = fmt.Sprintf("unknown: %+v", f.Type)
				}
			} else {
				typeName = fmt.Sprintf("unknown: %+v", f.Type)
			}
		default:
			typeName = fmt.Sprintf("unknown: %+v", f.Type)
		}

		if len(f.Names) == 1 {
			name = f.Names[0].Name
		} else if len(f.Names) == 0 {
			name = typeName
			isCurrentEmbed = true
		}

		result = append(result, &Field{
			Name:       name,
			Type:       typeName,
			ParentPath: parentName,
			IsEmbed:    isEmbed,
			IsPointer:  isPointer,
		})

		t, ok := f.Type.(*ast.Ident)
		if !ok {
			continue
		}

		if t.Obj != nil {
			if t.Obj.Decl == nil {
				continue
			}
			d, ok := t.Obj.Decl.(*ast.TypeSpec)
			if !ok {
				continue
			}
			s, ok := d.Type.(*ast.StructType)
			if !ok {
				continue
			}
			parentNameArg := d.Name.Name
			if len(parentName) > 0 {
				parentNameArg = fmt.Sprintf("%s.%s", parentName, parentNameArg)
			}

			fs := listAllField(s.Fields, parentNameArg, isCurrentEmbed)
			result = append(result, fs...)
		}
	}

	return result
}

func searchMetaProperties(fields []*Field) ([]*Field, error) {
	targetsMap := map[string]*MetaField{
		"CreatedAt": {
			Require:     true,
			RequireType: "time.Time",
		},
		"CreatedBy": {
			Require:     false,
			RequireType: "string",
		},
		"UpdatedAt": {
			Require:     true,
			RequireType: "time.Time",
		},
		"UpdatedBy": {
			Require:     false,
			RequireType: "string",
		},
		"DeletedAt": {
			Require:          true,
			RequireType:      "time.Time",
			RequireIsPointer: true,
		},
		"DeletedBy": {
			Require:     false,
			RequireType: "string",
		},
		"Version": {
			Require:     true,
			RequireType: "int",
		},
	}

	res := make([]*Field, 0, len(targetsMap))

	for _, f := range fields {
		if m, ok := targetsMap[f.Name]; ok {
			res = append(res, f)
			m.Find = true
			m.FindType = f.Type
			m.FindIsPointer = f.IsPointer
		}
	}

	for filedName, t := range targetsMap {
		if !t.Find && t.Require {
			return nil, xerrors.Errorf("%s is require", filedName)
		}
		if t.Find && t.RequireType != t.FindType {
			return nil, xerrors.Errorf("%s must be type %s", filedName, t.RequireType)
		}
		if t.Find && t.RequireIsPointer != t.FindIsPointer {
			p := "pointer"
			if !t.RequireIsPointer {
				p = "not pointer"
			}
			return nil, xerrors.Errorf("%s must be %s", filedName, p)
		}
	}

	return res, nil
}
