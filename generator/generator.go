package generator

import (
	"strings"

	go2tsparser "github.com/go-generalize/go2ts/pkg/parser"
	go2tstypes "github.com/go-generalize/go2ts/pkg/types"
	"golang.org/x/xerrors"
)

// Generator generates firestore CRUD functions
type Generator struct {
	dir   string
	types map[string]go2tstypes.Type
}

func NewGenerator(dir string) (*Generator, error) {
	psr, err := go2tsparser.NewParser(dir, func(fo *go2tsparser.FilterOpt) bool {
		return fo.BasePackage
	})

	if err != nil {
		return nil, xerrors.Errorf("failed to initializer go2ts parser: %w", err)
	}
	psr.Replacer = replacer

	types, err := psr.Parse()

	if err != nil {
		return nil, xerrors.Errorf("failed to parse with go2ts parser: %w", err)
	}

	return &Generator{
		dir:   dir,
		types: types,
	}, nil
}

// GenerateOption is a paramter to generate repository
type GenerateOption struct {
	OutputDir      string
	PackageName    string
	MockGenPath    string
	MockOutputPath string
	UseMetaField   bool
	Subcollection  bool
}

// NewDefaultGenerateOption returns a default GenerateOption
func NewDefaultGenerateOption() GenerateOption {
	return GenerateOption{
		OutputDir:      ".",
		MockGenPath:    "mockgen",
		MockOutputPath: "mock/mock_{{ .GeneratedFileName }}/mock_{{ .GeneratedFileName }}.go",
		UseMetaField:   true,
	}
}

func (g *Generator) Generate(structName string, opt GenerateOption) error {
	var typ *go2tstypes.Object
	for k, v := range g.types {
		t := strings.SplitN(k, ".", 2)[1]

		if t == structName {
			t, ok := v.(*go2tstypes.Object)

			if !ok {
				return xerrors.Errorf("Only struct is allowed")
			}
			typ = t
		}
	}

	if typ == nil {
		return xerrors.Errorf("struct not found: %s", structName)
	}

	gen, err := newStructGenerator(typ, structName, opt)

	if err != nil {
		return xerrors.Errorf("failed to initialize generator: %w", err)
	}

	if err := gen.parseType(); err != nil {
		return xerrors.Errorf("failed to parse type: %w", err)
	}

	if err := gen.generate(); err != nil {
		return xerrors.Errorf("failed to generate files: %w", err)
	}

	return nil
}
