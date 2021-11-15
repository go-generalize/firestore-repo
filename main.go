package main

import (
	"flag"
	"fmt"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/fatih/structtag"
	go2tsparser "github.com/go-generalize/go2ts/pkg/parser"
	go2tstypes "github.com/go-generalize/go2ts/pkg/types"
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
	packageName     = flag.String("p", "", "Specify the package name, default is the same as the original package")
	mockGenPath     = flag.String("mockgen", "mockgen", "Specify mockgen path")
	mockOutputPath  = flag.String("mock-output", defaultMockOut, "Specify directory to generate mock code in")
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
	if cont.Contains(reservedStructs, structName) {
		return xerrors.Errorf("reserved struct names cannot be used: %s(reserved: %v)", structName, reservedStructs)
	}

	disableMeta = &isDisableMeta
	isSubCollection = &subCollection

	psr, err := go2tsparser.NewParser(".", func(fo *go2tsparser.FilterOpt) bool {
		return fo.BasePackage && fo.Name == structName
	})

	if err != nil {
		return xerrors.Errorf("failed to initializer go2ts parser: %w", err)
	}
	psr.Replacer = replacer

	types, err := psr.Parse()

	if err != nil {
		return xerrors.Errorf("failed to parse with go2ts parser: %w", err)
	}

	if len(types) != 1 {
		return xerrors.Errorf("The number of parsed type should be 1, but got %d: %v", len(types), types)
	}

	var tstype *go2tstypes.Object
	for _, v := range types {
		tstype = v.(*go2tstypes.Object)
	}

	return traverse(tstype, structName)
}

func traverse(tstype *go2tstypes.Object, structName string) error {
	gen := &generator{
		PackageName: func() string {
			pn := *packageName
			if pn == "" {
				return tstype.PkgName
			}
			return pn
		}(),
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

	name := tstype.Position.Filename

	gen.FileName = strings.TrimSuffix(filepath.Base(name), ".go")
	gen.GeneratedFileName = gen.FileName + "_gen"

	gen.StructName = structName
	gen.StructNameRef = structName
	if !isCurrentDir {
		gen.StructNameRef = "model." + structName
		gen.ModelImportPath = importPath
	}

	return generate(gen, tstype)
}

func removeEmpty(arr []string) []string {
	ret := make([]string, 0, len(arr))

	for i := range arr {
		if arr[i] == "" {
			continue
		}

		ret = append(ret, arr[i])
	}

	return ret
}

func generateWithTSTypes(rawKey, firestoreKey string, gen *generator, obj *go2tstypes.Object, dupMap map[string]int) {
	entries := make([]go2tstypes.ObjectEntry, 0, len(obj.Entries))
	for _, e := range obj.Entries {
		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].FieldIndex < entries[j].FieldIndex
	})

	for _, e := range entries {
		typeName := getGo2tsType(e.Type)
		pos := e.Position.String()

		if typeName == "" {
			obj := e.Type.(*go2tstypes.Object)

			rawKey = strings.Join(removeEmpty([]string{rawKey, e.RawName}), ".")

			tags, err := structtag.Parse(e.RawTag)
			if err != nil {
				firestoreKey = strings.Join(removeEmpty([]string{firestoreKey, e.RawName}), ".")
			} else if t, err := tags.Get("firestore"); err != nil {
				firestoreKey = strings.Join(removeEmpty([]string{firestoreKey, e.RawName}), ".")
			} else {
				firestoreKey = strings.Join(removeEmpty([]string{firestoreKey, t.Name}), ".")
			}

			generateWithTSTypes(rawKey, firestoreKey, gen, obj, dupMap)
			continue
		}

		if !cont.Contains(supportType, typeName) {
			obj := strings.TrimPrefix(typeName, typeMap)

			if !cont.Contains(supportType, obj) {
				log.Printf(
					"%s: the type of `%s` is an invalid type in struct `%s` [%s]\n",
					pos, e.RawName, gen.StructName, typeName,
				)
				continue
			}
		}

		if strings.HasPrefix(typeName, "[]") {
			gen.SliceExist = true
		}

		if e.RawTag == "" {
			fieldInfo := &FieldInfo{
				FsTag:     strings.Join(removeEmpty([]string{firestoreKey, e.RawName}), "."),
				Field:     strings.Join(removeEmpty([]string{rawKey, e.RawName}), "."),
				FieldType: typeName,
				Indexes:   make([]*IndexesInfo, 0),
			}
			if _, err := appendIndexer(nil, firestoreKey, fieldInfo, dupMap); err != nil {
				log.Fatalf("%s: %v", pos, err)
			}
			gen.FieldInfos = append(gen.FieldInfos, fieldInfo)
			continue
		}

		tags, err := structtag.Parse(e.RawTag)
		if err != nil {
			log.Printf(
				"%s: tag for %s in struct %s in %s",
				pos, e.RawTag, gen.StructName, gen.GeneratedFileName+".go",
			)
			continue
		}

		if isIgnore(tags) {
			continue
		}

		if rawKey == "" && e.RawName == "Indexes" && typeName == typeBoolMap {
			gen.EnableIndexes = true
			fieldInfo := &FieldInfo{
				FsTag:     e.RawName,
				Field:     e.RawName,
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
				FsTag:     strings.Join(removeEmpty([]string{firestoreKey, e.RawName}), "."),
				Field:     strings.Join(removeEmpty([]string{rawKey, e.RawName}), "."),
				FieldType: typeName,
				Indexes:   make([]*IndexesInfo, 0),
			}
			if _, err = tags.Get("unique"); err == nil {
				if typeName != typeString {
					log.Fatalf("%s: The only field type that uses the `unique` tag is a string", pos)
				}
				fieldInfo.IsUnique = true
			}
			if fieldInfo, err = appendIndexer(tags, firestoreKey, fieldInfo, dupMap); err != nil {
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

		if err = keyFieldHandler(gen, tags, e.RawName, typeName); err != nil {
			log.Fatalf("%s: %v", pos, err)
		}
	}
}

func getGo2tsType(t go2tstypes.Type) string {
	switch t := t.(type) {
	case *go2tstypes.String:
		return "string"
	case *go2tstypes.Number:
		switch t.RawType {
		case types.Int:
			return "int"
		case types.Int8:
			return "int8"
		case types.Int16:
			return "int16"
		case types.Int32:
			return "int32"
		case types.Int64:
			return "int64"
		case types.Uint:
			return "uint"
		case types.Uint8:
			return "uint8"
		case types.Uint16:
			return "uint16"
		case types.Uint32:
			return "uint32"
		case types.Uint64:
			return "uint64"
		case types.Uintptr:
			return "uintptr"
		case types.Float32:
			return "float32"
		case types.Float64:
			return "float64"
		}
	case *go2tstypes.Boolean:
		return "bool"
	case *go2tstypes.Nullable:
		r := getGo2tsType(t.Inner)

		if strings.HasPrefix(r, "[]") {
			return r
		}

		return "*" + r
	case *go2tstypes.Array:
		return "[]" + getGo2tsType(t.Inner)
	case *go2tstypes.Date:
		return "time.Time"
	case *go2tstypes.Object:
		return ""
	case *go2tstypes.Map:
		return "map[" + getGo2tsType(t.Key) + "]" + getGo2tsType(t.Value)
	case *documentRef:
		return typeReference
	case *latLng:
		return typeLatLng
	}

	panic("unsupported: " + reflect.TypeOf(t).String())
}

func generate(gen *generator, tstype *go2tstypes.Object) error {
	dupMap := make(map[string]int)
	fieldLabel = gen.StructName + indexLabel

	gen.MetaFieldsEnabled = !*disableMeta

	rawEntries := map[string]go2tstypes.ObjectEntry{}
	for _, v := range tstype.Entries {
		rawEntries[v.RawName] = v
	}

	generateWithTSTypes("", "", gen, tstype, dupMap)

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

		gen.generateByFileName(fp, "label.go.tmpl")
	}

	{
		fp, err := os.Create(filepath.Join(*outputDir, "constant_gen.go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generateByFileName(fp, "constant.go.tmpl")
	}

	{
		fp, err := os.Create(filepath.Join(*outputDir, "errors_gen.go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generateByFileName(fp, "errors.go.tmpl")
	}

	{
		fp, err := os.Create(filepath.Join(*outputDir, "misc_gen.go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generateByFileName(fp, "misc.go.tmpl")
	}

	{
		fp, err := os.Create(filepath.Join(*outputDir, "query_builder_gen.go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generateByFileName(fp, "query_builder.go.tmpl")
	}

	{
		fp, err := os.Create(filepath.Join(*outputDir, "query_chain_gen.go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generateByFileName(fp, "query_chainer.go.tmpl")
	}

	{
		fp, err := os.Create(filepath.Join(*outputDir, "unique_gen.go"))
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		gen.generateByFileName(fp, "unique.go.tmpl")
	}

	return nil
}

func isIgnore(tags *structtag.Tags) bool {
	fsTag, err := tags.Get("firestore")
	if err != nil {
		return false
	}

	if _, err = tags.Get("firestore_key"); err == nil {
		return false
	}

	return strings.Split(fsTag.Value(), ",")[0] == "-"
}

func keyFieldHandler(gen *generator, tags *structtag.Tags, name, typeName string) error {
	fsTag, err := tags.Get("firestore")

	// firestore タグが存在しないか-になっていない
	if err != nil || strings.Split(fsTag.Value(), ",")[0] != "-" {
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

func appendIndexer(tags *structtag.Tags, fsTagBase string, fieldInfo *FieldInfo, dupMap map[string]int) (*FieldInfo, error) {
	filters := make([]string, 0)
	if tags != nil {
		if tag, err := fireStoreTagCheck(tags); err != nil {
			return nil, err
		} else if tag != "" {
			fieldInfo.FsTag = tag
			if fsTagBase != "" {
				fieldInfo.FsTag = fsTagBase + "." + tag
			}
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
			ConstName: strings.ReplaceAll(fieldLabel+fieldInfo.Field+strcase.ToCamel(patterns[i]), ".", "_"),
			Label:     uppercaseExtraction(fieldInfo.Field, dupMap),
			Method:    "Add",
		}

		switch patterns[i] {
		case prefix:
			idx.Use = isUseIndexer(filters, "p", prefix)
			idx.Method += strcase.ToCamel(prefix)
			idx.Comment = fmt.Sprintf("prefix-match of %s", fieldInfo.Field)
		case suffix:
			idx.Use = isUseIndexer(filters, "s", suffix)
			idx.Method += strcase.ToCamel(suffix)
			idx.Comment = fmt.Sprintf("suffix-match of %s", fieldInfo.Field)
		case like:
			idx.Use = isUseIndexer(filters, "l", like)
			idx.Method += biunigrams
			idx.Comment = fmt.Sprintf("like-match of %s", fieldInfo.Field)
		case equal:
			idx.Use = isUseIndexer(filters, "e", equal)
			idx.Comment = fmt.Sprintf("perfect-match of %s", fieldInfo.Field)
		}

		if fieldInfo.FieldType != typeString {
			idx.Method = "AddSomething"
		}

		fieldInfo.Indexes = append(fieldInfo.Indexes, idx)
	}

	sort.Slice(fieldInfo.Indexes, func(i, j int) bool {
		return fieldInfo.Indexes[i].Method < fieldInfo.Indexes[j].Method
	})

	return fieldInfo, nil
}

func fireStoreTagCheck(tags *structtag.Tags) (string, error) {
	fsTag, err := tags.Get("firestore")
	if err != nil {
		return "", nil
	}

	tag := strings.Split(fsTag.Value(), ",")[0]
	if !valueCheck.MatchString(tag) {
		return "", xerrors.New("key field for firestore should have other than blanks and symbols tag")
	}

	if unicode.IsDigit(rune(tag[0])) {
		return "", xerrors.New("key field for firestore should have indexerPrefix other than numbers required")
	}

	return tag, nil
}
