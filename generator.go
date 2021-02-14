package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"

	_ "github.com/go-generalize/firestore-repo/statik"
	"github.com/go-utils/cont"
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
	Use       bool
	Space1    string
	Space2    string
}

type FieldInfo struct {
	FsTag      string
	Field      string
	FieldType  string
	IsUnique   bool
	Space      string
	IndexerTag string
	Indexes    []*IndexesInfo
}

type generator struct {
	AppVersion        string
	PackageName       string
	ImportName        string
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
	IsSubCollection     bool

	MetaFields   map[string]*Field
	OmitMetaName string
}

func (g *generator) setting() {
	g.AppVersion = AppVersion
	g.RepositoryInterfaceName = g.StructName + "Repository"
	g.RepositoryStructName = strcase.ToLowerCamel(g.RepositoryInterfaceName)
	g.insertSpace()
}

func (g *generator) insertSpace() {
	var max int
	for _, x := range g.FieldInfos {
		if size := len(x.Field); size > max {
			max = size
		}
	}

	if len(g.MetaFields) > 0 {
		for k := range g.MetaFields {
			if size := len(k); size > max {
				max = size
			}
		}
		for k, v := range g.MetaFields {
			v.Space = strings.Repeat(" ", max-len(k))
		}
	}

	for _, x := range g.FieldInfos {
		x.Space = strings.Repeat(" ", max-len(x.Field))
	}
}

func (g *generator) insertSpaceForLabel() {
	var max1, max2 int
	for _, x := range g.FieldInfos {
		for _, index := range x.Indexes {
			if size := len(index.ConstName); size > max1 {
				max1 = size
			}
			if size := len(index.Label); size > max2 {
				max2 = size
			}
		}
	}
	for _, x := range g.FieldInfos {
		for _, index := range x.Indexes {
			index.Space1 = strings.Repeat(" ", max1-len(index.ConstName))
			index.Space2 = strings.Repeat(" ", max2-len(index.Label))
		}
	}
}

func (g *generator) generate(writer io.Writer) {
	g.setting()
	funcMap := g.setFuncMap()
	contents := getFileContents("gen")

	t := template.Must(template.New("Template").Funcs(funcMap).Parse(contents))

	if err := t.Execute(writer, g); err != nil {
		log.Printf("failed to execute template: %+v", err)
	}
}

func (g *generator) generateLabel(writer io.Writer) {
	g.insertSpaceForLabel()
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

func (g *generator) generateQueryBuilder(writer io.Writer) {
	contents := getFileContents("query_builder")

	t := template.Must(template.New("TemplateQueryBuilder").Parse(contents))

	if err := t.Execute(writer, g); err != nil {
		log.Printf("failed to execute template: %+v", err)
	}
}

func (g *generator) generateQueryChainer(writer io.Writer) {
	contents := getFileContents("query_chainer")

	t := template.Must(template.New("TemplateQueryChainer").Parse(contents))

	if err := t.Execute(writer, g); err != nil {
		log.Printf("failed to execute template: %+v", err)
	}
}

func (g *generator) generateUnique(writer io.Writer) {
	contents := getFileContents("unique")

	t := template.Must(template.New("TemplateUnique").Parse(contents))

	if err := t.Execute(writer, g); err != nil {
		log.Printf("failed to execute template: %+v", err)
	}
}

func (g *generator) metaJudgment() string {
	opts := "_"
	if len(g.MetaFields) > 0 {
		opts = "opts"
	}
	return opts
}

func (g *generator) setFuncMap() template.FuncMap {
	return template.FuncMap{
		"MetaJudgment": func() string {
			return g.metaJudgment()
		},
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
		"HasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(s, suffix)
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
		"IndexerInfo": func(fieldInfo *FieldInfo) (comment string) {
			if fieldInfo.IndexerTag == "" {
				return
			}
			comment += fmt.Sprintf(`// The value of the "indexer" tag = "%s"`, fieldInfo.IndexerTag)
			items := make([]string, 0)
			for _, index := range fieldInfo.Indexes {
				if !index.Use {
					continue
				}
				if !cont.Contains(items, index.Method) {
					items = append(items, index.Method)
				}
			}
			if len(items) > 3 {
				comment += "\n\t\t\t// "
				comment += strings.Join(items, "/")
				comment += " is valid."
			}
			return
		},
		"GetFunc": func() string {
			raw := fmt.Sprintf(
				"Get(ctx context.Context, %s %s, opts ...GetOption) (*%s, error)",
				g.KeyValueName, g.KeyFieldType, g.StructName,
			)
			return raw
		},
		"GetWithDocFunc": func() string {
			raw := fmt.Sprintf(
				"GetWithDoc(ctx context.Context, doc *firestore.DocumentRef, opts ...GetOption) (*%s, error)",
				g.StructName,
			)
			return raw
		},
		"InsertFunc": func() string {
			return fmt.Sprintf("Insert(ctx context.Context, subject *%s) (_ %s, err error)", g.StructName, g.KeyFieldType)
		},
		"UpdateFunc": func() string {
			return fmt.Sprintf("Update(ctx context.Context, subject *%s) (err error)", g.StructName)
		},
		"StrictUpdateFunc": func() string {
			return fmt.Sprintf(
				"StrictUpdate(ctx context.Context, id string, param *%sUpdateParam, opts ...firestore.Precondition) error",
				g.StructName,
			)
		},
		"DeleteFunc": func() string {
			return fmt.Sprintf("Delete(ctx context.Context, subject *%s, opts ...DeleteOption) (err error)", g.StructName)
		},
		"DeleteByFunc": func() string {
			raw := fmt.Sprintf(
				"DeleteBy%s(ctx context.Context, %s %s, opts ...DeleteOption) (err error)",
				g.KeyFieldName, g.KeyValueName, g.KeyFieldType,
			)
			return raw
		},
		"GetMultiFunc": func() string {
			raw := fmt.Sprintf(
				"GetMulti(ctx context.Context, %s []%s, opts ...GetOption) ([]*%s, error)",
				plural.Convert(g.KeyValueName), g.KeyFieldType, g.StructName,
			)
			return raw
		},
		"InsertMultiFunc": func() string {
			return fmt.Sprintf("InsertMulti(ctx context.Context, subjects []*%s) (_ []%s, er error)", g.StructName, g.KeyFieldType)
		},
		"UpdateMultiFunc": func() string {
			return fmt.Sprintf("UpdateMulti(ctx context.Context, subjects []*%s) (er error)", g.StructName)
		},
		"DeleteMultiFunc": func() string {
			return fmt.Sprintf("DeleteMulti(ctx context.Context, subjects []*%s, opts ...DeleteOption) (er error)", g.StructName)
		},
		"DeleteMultiByFunc": func() string {
			raw := fmt.Sprintf(
				"DeleteMultiBy%s(ctx context.Context, %s []%s, opts ...DeleteOption) (er error)",
				plural.Convert(g.KeyFieldName), plural.Convert(g.KeyValueName), g.KeyFieldType,
			)
			return raw
		},
		"ListFunc": func() string {
			return fmt.Sprintf(
				"List(ctx context.Context, req *%sListReq, q *firestore.Query) ([]*%s, error)",
				g.StructName, g.StructName)
		},
		"GetWithTxFunc": func() string {
			raw := fmt.Sprintf(
				"GetWithTx(tx *firestore.Transaction, %s %s, opts ...GetOption) (*%s, error)",
				g.KeyValueName, g.KeyFieldType, g.StructName,
			)
			return raw
		},
		"GetWithDocWithTxFunc": func() string {
			raw := fmt.Sprintf(
				"GetWithDocWithTx(tx *firestore.Transaction, doc *firestore.DocumentRef, opts ...GetOption) (*%s, error)",
				g.StructName,
			)
			return raw
		},
		"InsertWithTxFunc": func() string {
			return fmt.Sprintf(
				"InsertWithTx(ctx context.Context, tx *firestore.Transaction, subject *%s) (_ %s, err error)",
				g.StructName, g.KeyFieldType,
			)
		},
		"UpdateWithTxFunc": func() string {
			return fmt.Sprintf("UpdateWithTx(ctx context.Context, tx *firestore.Transaction, subject *%s) (err error)", g.StructName)
		},
		"StrictUpdateWithTxFunc": func() string {
			return fmt.Sprintf(
				"StrictUpdateWithTx(tx *firestore.Transaction, id string, param *%sUpdateParam, opts ...firestore.Precondition) error",
				g.StructName,
			)
		},
		"DeleteWithTxFunc": func() string {
			return fmt.Sprintf(
				"DeleteWithTx(ctx context.Context, tx *firestore.Transaction, subject *%s, opts ...DeleteOption) (err error)",
				g.StructName,
			)
		},
		"DeleteByWithTxFunc": func() string {
			return fmt.Sprintf(
				"DeleteBy%sWithTx(ctx context.Context, tx *firestore.Transaction, %s %s, opts ...DeleteOption) (err error)",
				g.KeyFieldName, g.KeyValueName, g.KeyFieldType,
			)
		},
		"ListWithTxFunc": func() string {
			return fmt.Sprintf(
				"ListWithTx(tx *firestore.Transaction, req *%sListReq, q *firestore.Query) ([]*%s, error)",
				g.StructName, g.StructName)
		},
		"GetMultiWithTxFunc": func() string {
			return fmt.Sprintf(
				"GetMultiWithTx(tx *firestore.Transaction, %s []%s, opts ...GetOption) ([]*%s, error)",
				plural.Convert(g.KeyValueName), g.KeyFieldType, g.StructName,
			)
		},
		"InsertMultiWithTxFunc": func() string {
			return fmt.Sprintf(
				"InsertMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*%s) (_ []string, er error)",
				g.StructName,
			)
		},
		"UpdateMultiWithTxFunc": func() string {
			return fmt.Sprintf("UpdateMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*%s) (er error)", g.StructName)
		},
		"DeleteMultiWithTxFunc": func() string {
			return fmt.Sprintf("DeleteMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*%s, opts ...DeleteOption) (er error)", g.StructName)
		},
		"DeleteMultiByWithTxFunc": func() string {
			raw := fmt.Sprintf(
				"DeleteMultiBy%sWithTx(ctx context.Context, tx *firestore.Transaction, %s []%s, opts ...DeleteOption) (er error)",
				plural.Convert(g.KeyFieldName), plural.Convert(g.KeyValueName), g.KeyFieldType,
			)
			return raw
		},
	}
}
