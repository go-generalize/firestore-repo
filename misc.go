package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

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
		typeTimePtr,
		"*" + typeLatLng,
		"*" + typeReference,
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
