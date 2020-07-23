package main

import (
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"

	_ "github.com/go-generalize/firestore-repo/statik"
	"github.com/go-utils/plural"
	"github.com/iancoleman/strcase"
	"github.com/rakyll/statik/fs"
)

var statikFS http.FileSystem

func init() {
	var err error
	statikFS, err = fs.New()
	if err != nil {
		log.Fatal(err)
	}
}

type IndexesInfo struct {
	Comment   string
	ConstName string
	Label     string
	Method    string
}

type FieldInfo struct {
	FsTag     string
	Field     string
	FieldType string
	Operator  Operator
	Space     string
	Indexes   []*IndexesInfo
}

type ImportInfo struct {
	Name string
}

type generator struct {
	PackageName       string
	ImportName        string
	ImportList        []ImportInfo
	GeneratedFileName string
	FileName          string
	StructName        string

	RepositoryStructName    string
	RepositoryInterfaceName string

	KeyFieldName string
	KeyFieldType string
	KeyValueName string // lower camel case

	FieldInfos []*FieldInfo

	EnableIndexes       bool
	FieldInfoForIndexes *FieldInfo
	BoolCriteriaCnt     int
	SliceExist          bool

	AutomaticGeneration bool

	MetaFields map[string]Field
}

func (g *generator) setting() {
	g.RepositoryInterfaceName = g.StructName + "Repository"
	g.RepositoryStructName = strcase.ToLowerCamel(g.RepositoryInterfaceName)
	g.buildConditions()
	g.insertSpace()
}

func (g *generator) buildConditions() {
	dedupe := make(map[string]bool)
	for _, field := range g.FieldInfos {
		ft := field.FieldType
		switch ft {
		case "time.Time":
			if _, ok := dedupe[ft]; !ok {
				dedupe[ft] = true
				g.ImportList = append(g.ImportList, ImportInfo{strings.Split(ft, ".")[0]})
			}
		}
	}
}

func (g *generator) insertSpace() {
	max := 0
	for _, x := range g.FieldInfos {
		if len(x.Field) > max {
			max = len(x.Field)
		}
	}

	for _, x := range g.FieldInfos {
		x.Space = strings.Repeat(" ", max-len(x.Field))
	}
}

func (g *generator) generate(writer io.Writer) {
	g.setting()
	funcMap := setFuncMap()
	contents := getFileContents("gen")

	t := template.Must(template.New("Template").Funcs(funcMap).Parse(contents))

	if err := t.Execute(writer, g); err != nil {
		log.Printf("failed to execute template: %+v", err)
	}
}

func (g *generator) generateLabel(writer io.Writer) {
	contents := getFileContents("label")

	t := template.Must(template.New("TemplateLabel").Parse(contents))

	if err := t.Execute(writer, g); err != nil {
		log.Printf("failed to execute template: %+v", err)
	}
}

func (g *generator) generateConstant(writer io.Writer) {
	contents := getFileContents("constant")

	t := template.Must(template.New("TemplateConstant").Parse(contents))

	if err := t.Execute(writer, g); err != nil {
		log.Printf("failed to execute template: %+v", err)
	}
}

func (g *generator) generateMisc(writer io.Writer) {
	contents := getFileContents("misc")

	t := template.Must(template.New("TemplateMisc").Parse(contents))

	if err := t.Execute(writer, g); err != nil {
		log.Printf("failed to execute template: %+v", err)
	}
}

func (g *generator) generateQuery(writer io.Writer) {
	contents := getFileContents("query")

	t := template.Must(template.New("TemplateQuery").Parse(contents))

	if err := t.Execute(writer, g); err != nil {
		log.Printf("failed to execute template: %+v", err)
	}
}

func setFuncMap() template.FuncMap {
	return template.FuncMap{
		"Parse": func(fieldType string) string {
			fieldType = strings.TrimPrefix(fieldType, "[]")
			fn := "Int"
			switch fieldType {
			case typeInt:
			case typeInt64:
				fn = "Int64"
			case typeFloat64:
				fn = "Float64"
			case typeString:
				fn = "String"
			case typeBool:
				fn = "Bool"
			default:
				panic("invalid types")
			}
			return fn
		},
		"HasSlice": func(types string) bool {
			return strings.HasPrefix(types, "[]")
		},
		"HasMap": func(types string) bool {
			return strings.HasPrefix(types, typeMap)
		},
		"PluralForm": func(word string) string {
			return plural.Convert(word)
		},
	}
}
