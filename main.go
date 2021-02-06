package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/fatih/structtag"
	"github.com/go-utils/cont"
	"github.com/iancoleman/strcase"
	"golang.org/x/xerrors"
)

func init() {
	for _, x := range supportType {
		supportType = append(supportType, "[]"+x)
	}
}

var (
	isShowVersion   = flag.Bool("v", false, "print version")
	isSubCollection = flag.Bool("sub-collection", false, "is SubCollection")
	disableMeta     = flag.Bool("disable-meta", false, "Disable meta embed for Lock")
)

func main() {
	flag.Parse()

	if *isShowVersion {
		fmt.Printf("Firestore Model Generator: %s\n", AppVersion)
		return
	}

	l := flag.NArg()
	if l < 1 {
		fmt.Println("You have to specify the struct name of target")
		os.Exit(1)
	}

	if err := run(flag.Arg(0), *disableMeta, *isSubCollection); err != nil {
		log.Fatal(err.Error())
	}
}

func run(structName string, isDisableMeta, subCollection bool) error {
	disableMeta = &isDisableMeta
	isSubCollection = &subCollection
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, ".", nil, parser.AllErrors)

	if err != nil {
		panic(err)
	}

	for name, v := range pkgs {
		if strings.HasSuffix(name, "_test") {
			continue
		}

		return traverse(v, fs, structName)
	}

	return nil
}

func traverse(pkg *ast.Package, fs *token.FileSet, structName string) error {
	gen := &generator{PackageName: pkg.Name}
	if *isSubCollection {
		gen.IsSubCollection = true
	}

	for name, file := range pkg.Files {
		gen.FileName = strings.TrimSuffix(filepath.Base(name), ".go")
		gen.GeneratedFileName = gen.FileName + "_gen"

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			if genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				// 型定義
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				name := typeSpec.Name.Name

				if name != structName {
					continue
				}

				// structの定義
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				gen.StructName = name

				return generate(gen, fs, structType)
			}
		}
	}

	return xerrors.Errorf("no such struct: %s", structName)
}

func generate(gen *generator, fs *token.FileSet, structType *ast.StructType) error {
	dupMap := make(map[string]int)
	fieldLabel = gen.StructName + queryLabel

	metaList := make(map[string]*Field)
	metaFieldName := ""
	if !*disableMeta {
		fList := listAllField(structType.Fields, "", false)
		for _, field := range fList {
			if gen.OmitMetaParentName != "" {
				break
			}
			if sp := strings.Split(field.Name, "."); len(sp) > 1 {
				gen.OmitMetaParentName = sp[0]
			}
		}

		metas, mfn, err := searchMetaProperties(fList)
		if err != nil {
			return err
		}

		metaFieldName = mfn

		for _, m := range metas {
			metaList[m.Name] = m
		}
	}
	gen.MetaFields = metaList

	for _, field := range structType.Fields.List {
		// structの各fieldを調査
		if len(field.Names) > 1 {
			return xerrors.New("`field.Names` must have only one element")
		}

		isMetaFiled := false
		name := ""

		if field.Names == nil || len(field.Names) == 0 {
			switch field.Type.(type) {
			case *ast.Ident:
				name = field.Type.(*ast.Ident).Name
			case *ast.SelectorExpr:
				name = field.Type.(*ast.SelectorExpr).Sel.Name
			}

			if !*disableMeta && name == metaFieldName {
				isMetaFiled = true
				gen.OmitMetaName = name
				gen.MetaPath = name
			}
		} else {
			name = field.Names[0].Name
		}

		pos := fs.Position(field.Pos()).String()

		typeName := getTypeName(field.Type)
		if !isMetaFiled && !cont.Contains(supportType, typeName) {
			typeNameDetail := getTypeNameDetail(field.Type)
			obj := strings.TrimPrefix(typeNameDetail, typeMap)

			if !cont.Contains(supportType, obj) {
				log.Printf(
					"%s: the type of `%s` is an invalid type in struct `%s` [%s]\n",
					pos, name, gen.StructName, typeName,
				)
				continue
			}
			typeName = typeNameDetail
		}

		if strings.HasPrefix(typeName, "[]") {
			gen.SliceExist = true
		}

		if field.Tag == nil {
			fieldInfo := &FieldInfo{
				FsTag:     name,
				Field:     name,
				FieldType: typeName,
				Indexes:   make([]*IndexesInfo, 0),
			}
			appendIndexesInfo(fieldInfo, dupMap)
			gen.FieldInfos = append(gen.FieldInfos, fieldInfo)
			continue
		}

		tags, err := structtag.Parse(strings.Trim(field.Tag.Value, "`"))
		if err != nil {
			log.Printf(
				"%s: tag for %s in struct %s in %s",
				pos, name, gen.StructName, gen.GeneratedFileName+".go",
			)
			continue
		}
		if name == "Indexes" {
			gen.EnableIndexes = true
			fieldInfo := &FieldInfo{
				FsTag:     name,
				Field:     name,
				FieldType: typeName,
			}

			if tag, er := fireStoreTagCheck(tags); er != nil {
				log.Fatalf("%s: %v", pos, er)
			} else if tag != "" {
				fieldInfo.FsTag = tag
			}

			gen.FieldInfoForIndexes = fieldInfo
			continue
		}

		tag, err := tags.Get("firestore_key")
		if err != nil {
			fieldInfo := &FieldInfo{
				FsTag:     name,
				Field:     name,
				FieldType: typeName,
				Indexes:   make([]*IndexesInfo, 0),
			}
			if fieldInfo, err = appendIndexer(tags, fieldInfo, dupMap); err != nil {
				log.Fatalf("%s: %v", pos, err)
			}
			gen.FieldInfos = append(gen.FieldInfos, fieldInfo)
			continue
		}

		switch tag.Value() {
		case "":
			// ok
		case "auto":
			gen.AutomaticGeneration = true
		default:
			log.Fatalf(
				`%s: The contents of the firestore_key tag should be "" or "auto"`, pos)
		}

		if err := keyFieldHandler(gen, tags, name, typeName); err != nil {
			log.Fatalf("%s: %v", pos, err)
		}
	}

	if gen.OmitMetaParentName != "" {
		gen.MetaPath = fmt.Sprintf("%s.%s", gen.OmitMetaParentName, gen.OmitMetaName)
	}

	{
		fileName := gen.GeneratedFileName + ".go"
		fp, err := os.Create(fileName)
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generate(fp)

		_, err = execCommand("goimports", "-w", fileName)
		if err != nil {
			log.Fatalf("goimports exec error (%s): %v", fileName, err)
		}
	}

	if gen.EnableIndexes {
		path := gen.FileName + "_label.go"
		fp, err := os.Create(path)
		if err != nil {
			panic(err)
		}
		defer fp.Close()
		gen.generateLabel(fp)
	}

	{
		fp, err := os.Create("constant_gen.go")
		if err != nil {
			panic(err)
		}
		defer fp.Close()
		gen.generateConstant(fp)
	}

	{
		fp, err := os.Create("misc_gen.go")
		if err != nil {
			panic(err)
		}
		defer fp.Close()
		gen.generateMisc(fp)
	}

	{
		fp, err := os.Create("query_builder_gen.go")
		if err != nil {
			panic(err)
		}
		defer fp.Close()
		gen.generateQueryBuilder(fp)
	}

	{
		fp, err := os.Create("query_chain_gen.go")
		if err != nil {
			panic(err)
		}
		defer fp.Close()
		gen.generateQueryChainer(fp)
	}

	return nil
}

func keyFieldHandler(gen *generator, tags *structtag.Tags, name, typeName string) error {
	FsTag, err := tags.Get("firestore")

	// firestore タグが存在しないか-になっていない
	if err != nil || strings.Split(FsTag.Value(), ",")[0] != "-" {
		return xerrors.New("key field for firestore should have firestore:\"-\" tag")
	}

	gen.KeyFieldName = name
	gen.KeyFieldType = typeName

	if gen.KeyFieldType != typeString {
		return xerrors.New("supported key types are string")
	}

	gen.KeyValueName = strcase.ToLowerCamel(name)
	return nil
}

func appendIndexer(tags *structtag.Tags, fieldInfo *FieldInfo, dupMap map[string]int) (*FieldInfo, error) {
	if tag, err := fireStoreTagCheck(tags); err != nil {
		return nil, err
	} else if tag != "" {
		fieldInfo.FsTag = tag
	}
	if idr, err := tags.Get("indexer"); err != nil || fieldInfo.FieldType != typeString {
		appendIndexesInfo(fieldInfo, dupMap)
	} else {
		filters := strings.Split(idr.Value(), ",")
		dupIdr := make(map[string]struct{})
		for _, fil := range filters {
			idx := &IndexesInfo{
				ConstName: fieldLabel + fieldInfo.Field,
				Label:     uppercaseExtraction(fieldInfo.Field, dupMap),
				Method:    "Add",
			}
			var dupFlag string
			switch fil {
			case "p", "prefix": // 前方一致 (AddPrefix)
				idx.Method += prefix
				idx.ConstName += prefix
				idx.Comment = fmt.Sprintf("%s %s前方一致", idx.ConstName, fieldInfo.Field)
				dupFlag = "p"
			case "s", "suffix": /* TODO 後方一致
				idx.Method += Suffix
				idx.ConstName += Suffix
				idx.Comment = fmt.Sprintf("%s %s後方一致", idx.ConstName, name)
				dup = "s"*/
			case "e", "equal": // 完全一致 (Add) Default
				idx.Comment = fmt.Sprintf("%s %s", idx.ConstName, fieldInfo.Field)
				dupIdr["equal"] = struct{}{}
				dupFlag = "e"
			case "l", "like": // 部分一致
				idx.Method += biunigrams
				idx.ConstName += "Like"
				idx.Comment = fmt.Sprintf("%s %s部分一致", idx.ConstName, fieldInfo.Field)
				dupFlag = "l"
			default:
				continue
			}
			if _, ok := dupIdr[dupFlag]; ok {
				continue
			}
			dupIdr[dupFlag] = struct{}{}
			fieldInfo.Indexes = append(fieldInfo.Indexes, idx)
		}
	}
	return fieldInfo, nil
}

func fireStoreTagCheck(tags *structtag.Tags) (string, error) {
	if FsTag, err := tags.Get("firestore"); err == nil {
		tag := strings.Split(FsTag.Value(), ",")[0]
		if !valueCheck.MatchString(tag) {
			return tag, xerrors.New("key field for firestore should have other than blanks and symbols tag")
		}
		if unicode.IsDigit(rune(tag[0])) {
			return "", xerrors.New("key field for firestore should have indexerPrefix other than numbers required")
		}
		return tag, nil
	}
	return "", nil
}
