package main

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"regexp"

	"github.com/go-utils/cont"
	"golang.org/x/xerrors"
)

var (
	fieldLabel  string
	valueCheck  = regexp.MustCompile("^[0-9a-zA-Z_]+$")
	supportType = []string{
		typeBool,
		typeString,
		typeInt,
		typeInt64,
		typeFloat64,
		typeTime,
		typeLatLng,
		typeReference,
		typeStringMap,
		typeIntMap,
		typeInt64Map,
		typeFloat64Map,
	}
	reservedStructs = []string{
		"Unique",
	}
)

func uppercaseExtraction(name string, dupMap map[string]int) (lower string) {
	defer func() {
		if _, ok := dupMap[lower]; ok {
			lower = fmt.Sprintf("%s%d", lower, dupMap[lower])
		}
	}()
	for i, x := range name {
		switch {
		case 65 <= x && x <= 90:
			x += 32
			fallthrough
		case 97 <= x && x <= 122:
			if i == 0 {
				lower += string(x)
			}
			if _, ok := dupMap[lower]; !ok {
				dupMap[lower] = 1
				return
			}

			if dupMap[lower] >= 9 && len(name) > i+1 {
				lower += string(name[i+1])
				continue
			}
			dupMap[lower]++
			return
		}
	}
	return
}

func getTypeName(typ ast.Expr) string {
	switch v := typ.(type) {
	case *ast.SelectorExpr:
		return getTypeName(v.X) + "." + v.Sel.Name

	case *ast.Ident:
		return v.Name

	case *ast.StarExpr:
		return "*" + getTypeName(v.X)

	case *ast.ArrayType:
		return "[]" + getTypeName(v.Elt)

	default:
		return ""
	}
}

func getTypeNameDetail(typ ast.Expr) string {
	switch v := typ.(type) {
	case *ast.SelectorExpr:
		return getTypeNameDetail(v.X) + "." + v.Sel.Name

	case *ast.Ident:
		name := v.Name
		if v.Obj != nil {
			upper := getTypeNameDetail(v.Obj.Decl.(*ast.TypeSpec).Type)
			if upper != "" {
				name = upper
			}
			switch v.Obj.Decl.(*ast.TypeSpec).Type.(type) {
			case *ast.StructType:
				name += "STRUCT"
				// TODO WIP support Struct (strings.HasSuffix(name, "STRUCT"))
			}
		}

		return name

	case *ast.StarExpr:
		x, ok := v.X.(*ast.Ident)
		name := getTypeNameDetail(v.X)
		if name == "" && ok {
			name = x.Name
		}

		return "*" + name

	case *ast.ArrayType:
		return "[]" + getTypeNameDetail(v.Elt)

	case *ast.MapType:
		name := "map[%s]"
		switch key := v.Key.(type) {
		case *ast.Ident:
			if key.Name == "string" {
				name = fmt.Sprintf(name, key.Name)
				break
			}
			name = fmt.Sprintf(name, NgType)
		default:
			name = fmt.Sprintf(name, NgType)
		}
		switch val := v.Value.(type) {
		case *ast.Ident:
			if cont.Contains(supportType, val.Name) {
				name += val.Name
				break
			}
			name += NgType
		case *ast.InterfaceType:
			name += "interface{}"
		default:
			name += NgType
		}

		return name

	default:
		return ""
	}
}

func isCurrentDirectory(path string) (bool, error) {
	abs, err := filepath.Abs(path)

	if err != nil {
		return false, xerrors.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	wd, err := os.Getwd()

	if err != nil {
		return false, xerrors.Errorf("failed to get working directory: %w", err)
	}

	return filepath.Clean(abs) == filepath.Clean(wd), nil
}
