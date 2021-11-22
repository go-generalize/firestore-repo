package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-generalize/firestore-repo/generator"
)

var (
	isShowVersion   = flag.Bool("v", false, "print version")
	isSubCollection = flag.Bool("sub-collection", false, "is SubCollection")
	disableMeta     = flag.Bool("disable-meta", false, "Disable meta embed")
	outputDir       = flag.String("o", "./", "Specify directory to generate code in")
	packageName     = flag.String("p", "", "Specify the package name, default is the same as the original package")
	mockGenPath     = flag.String("mockgen", "mockgen", "Specify mockgen path")
	mockOutputPath  = flag.String("mock-output", "mock/mock_{{ .GeneratedFileName }}/mock_{{ .GeneratedFileName }}.go", "Specify directory to generate mock code in")
)

func main() {
	if *isShowVersion {
		fmt.Printf("Firestore Model Generator: %s\n", AppVersion)
		return
	}

	l := flag.NArg()
	if l < 1 {
		fmt.Fprintln(os.Stderr, "You have to specify the struct name of target")
		os.Exit(1)
	}

	gen, err := generator.NewGenerator(".")

	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to initialize generator: %+v", err)
		os.Exit(1)
	}

	gen.Generate(flag.Arg(0), generator.GenerateOption{
		OutputDir:      *outputDir,
		PackageName:    *packageName,
		MockGenPath:    *mockGenPath,
		MockOutputPath: *mockOutputPath,
		UseMetaField:   !*disableMeta,
		Subcollection:  *isSubCollection,
	})
}
