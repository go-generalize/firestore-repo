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
	"sort"
	"strings"
	"unicode"

	"github.com/fatih/structtag"
	"github.com/go-utils/cont"
	"github.com/go-utils/gopackages"
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
	disableMeta     = flag.Bool("disable-meta", false, "Disable meta embed")
	outputDir       = flag.String("o", "./", "Specify directory to generate code in")
	mockGenPath     = flag.String("mockgen", "mockgen", "Specify mockgen path")
	mockOutputPath  = flag.String(
		"mock-output", defaultMockOut, "Specify directory to generate mock code in",
	)
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
	gen := &generator{
		PackageName: pkg.Name,
		MockGenPath: *mockGenPath,
	}
	if *isSubCollection {
		gen.IsSubCollection = true
	}

	isCurrentDir, err := isCurrentDirectory(*outputDir)

	if err != nil {
		return xerrors.Errorf("failed to call isCurrentDirectory: %w", err)
	}

	mod, err := gopackages.NewModule(".")

	if err != nil {
		return xerrors.Errorf("failed to initialize gopackages.Module: %w", err)
	}

	importPath, err := mod.GetImportPath(".")

	if err != nil {
		return xerrors.Errorf("failed to get import path for current directory: %w", err)
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

				if cont.Contains(reservedStructs, name) {
					log.Fatalf("%s is a reserved struct", name)
				}

				// structの定義
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				gen.StructName = name
				gen.StructNameRef = name
				if !isCurrentDir {
					gen.StructNameRef = "model." + name
					gen.ModelImportPath = importPath
				}

				return generate(gen, fs, structType)
			}
		}
	}

	return xerrors.Errorf("no such struct: %s", structName)
}

func generate(gen *generator, fs *token.FileSet, structType *ast.StructType) error {
	dupMap := make(map[string]int)
	fieldLabel = gen.StructName + indexLabel

	metaList := make(map[string]*Field)
	metaFieldName := ""
	if !*disableMeta {
		fList := listAllField(structType.Fields, "", false)

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
			if _, err := appendIndexer(nil, fieldInfo, dupMap); err != nil {
				log.Fatalf("%s: %v", pos, err)
			}
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
		if name == "Indexes" && typeName == typeBoolMap {
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
			if _, err = tags.Get("unique"); err == nil {
				if typeName != typeString {
					log.Fatalf("%s: The only field type that uses the `unique` tag is a string", pos)
				}
				fieldInfo.IsUnique = true
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

	{
		gen.MockOutputPath = func() string {
			mop := *mockOutputPath
			if mop == defaultMockOut {
				return strings.ReplaceAll(mop, "{{ .GeneratedFileName }}", gen.GeneratedFileName)
			}
			if !strings.HasSuffix(mop, ".go") {
				mop += ".go"
			}
			return mop
		}()
	}

	{
		fp, err := os.Create(filepath.Join(*outputDir, gen.GeneratedFileName+".go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generate(fp)
	}

	if gen.EnableIndexes {
		fp, err := os.Create(filepath.Join(*outputDir, gen.FileName+"_label_gen.go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generateLabel(fp)
	}

	{
		fp, err := os.Create(filepath.Join(*outputDir, "constant_gen.go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generateConstant(fp)
	}

	{
		fp, err := os.Create(filepath.Join(*outputDir, "misc_gen.go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generateMisc(fp)
	}

	{
		fp, err := os.Create(filepath.Join(*outputDir, "query_builder_gen.go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generateQueryBuilder(fp)
	}

	{
		fp, err := os.Create(filepath.Join(*outputDir, "query_chain_gen.go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generateQueryChainer(fp)
	}

	{
		fp, err := os.Create(filepath.Join(*outputDir, "unique_gen.go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generateUnique(fp)
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

func isUseIndexer(filters []string, p1, p2 string) bool {
	for _, filter := range filters {
		switch filter {
		case p1, p2:
			return true
		}
	}
	return false
}

func appendIndexer(tags *structtag.Tags, fieldInfo *FieldInfo, dupMap map[string]int) (*FieldInfo, error) {
	filters := make([]string, 0)
	if tags != nil {
		if tag, err := fireStoreTagCheck(tags); err != nil {
			return nil, err
		} else if tag != "" {
			fieldInfo.FsTag = tag
		}
		idr, err := tags.Get("indexer")
		if err == nil {
			fieldInfo.IndexerTag = idr.Value()
			filters = strings.Split(idr.Value(), ",")
		}
	}
	patterns := [4]string{
		prefix, suffix, like, equal,
	}
	for i := range patterns {
		idx := &IndexesInfo{
			ConstName: fieldLabel + fieldInfo.Field + patterns[i],
			Label:     uppercaseExtraction(fieldInfo.Field, dupMap),
			Method:    "Add",
		}
		if fieldInfo.FieldType != typeString {
			idx.Use = isUseIndexer(filters, "e", equal)
			idx.Method += "Something"
			fieldInfo.Indexes = append(fieldInfo.Indexes, idx)
			idx.Comment = fmt.Sprintf("perfect-match of %s", fieldInfo.Field)
			continue
		}
		switch patterns[i] {
		case prefix:
			idx.Use = isUseIndexer(filters, "p", prefix)
			idx.Method += prefix
			idx.Comment = fmt.Sprintf("prefix-match of %s", fieldInfo.Field)
		case suffix:
			idx.Use = isUseIndexer(filters, "s", suffix)
			idx.Method += suffix
			idx.Comment = fmt.Sprintf("suffix-match of %s", fieldInfo.Field)
		case like:
			idx.Use = isUseIndexer(filters, "l", like)
			idx.Method += biunigrams
			idx.Comment = fmt.Sprintf("like-match of %s", fieldInfo.Field)
		case equal:
			idx.Use = isUseIndexer(filters, "e", equal)
			idx.Comment = fmt.Sprintf("perfect-match of %s", fieldInfo.Field)
		}
		fieldInfo.Indexes = append(fieldInfo.Indexes, idx)
	}
	sort.Slice(fieldInfo.Indexes, func(i, j int) bool {
		return fieldInfo.Indexes[i].Method < fieldInfo.Indexes[j].Method
	})
	return fieldInfo, nil
}

func fireStoreTagCheck(tags *structtag.Tags) (string, error) {
	if FsTag, err := tags.Get("firestore"); err == nil {
		tag := strings.Split(FsTag.Value(), ",")[0]
		if !valueCheck.MatchString(tag) {
			return "", xerrors.New("key field for firestore should have other than blanks and symbols tag")
		}
		if unicode.IsDigit(rune(tag[0])) {
			return "", xerrors.New("key field for firestore should have indexerPrefix other than numbers required")
		}
		return tag, nil
	}
	return "", nil
}
