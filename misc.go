package main

import (
	"fmt"
	"go/ast"
	"io/ioutil"
	"log"
	"regexp"

	"github.com/go-utils/cont"
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
		typeStringMap,
		typeIntMap,
		typeInt64Map,
		typeFloat64Map,
	}
	supportOperators = []Operator{
		OperatorLessThan,
		OperatorLessThanOrEqual,
		OperatorGreaterThan,
		OperatorGreaterThanOrEqual,
		OperatorEqual,
		OperatorIn,
		OperatorArrayContains,
		OperatorArrayContainsAny,
	}
)

func getFileContents(name string) string {
	fp, err := statikFS.Open("/" + name + ".go.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()

	contents, err := ioutil.ReadAll(fp)
	if err != nil {
		log.Fatal(err)
	}

	return string(contents)
}

func uppercaseExtraction(name string, dupMap map[string]int) (lower string) {
	for _, x := range name {
		if 65 <= x && x <= 90 {
			lower += string(x + 32)
		}
	}
	if _, ok := dupMap[lower]; !ok {
		dupMap[lower] = 1
	} else {
		dupMap[lower]++
		lower = fmt.Sprintf("%s%d", lower, dupMap[lower])
	}
	return
}

func appendIndexesInfo(fieldInfo *FieldInfo, dupMap map[string]int) {
	idx := &IndexesInfo{
		ConstName: fieldLabel + fieldInfo.Field,
		Label:     uppercaseExtraction(fieldInfo.Field, dupMap),
		Method:    "Add",
	}
	idx.Comment = fmt.Sprintf("%s %s", idx.ConstName, fieldInfo.Field)
	if fieldInfo.FieldType != typeString {
		idx.Method += "Something"
	}
	fieldInfo.Indexes = append(fieldInfo.Indexes, idx)
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
