package main

import (
	"go/types"

	go2tstypes "github.com/go-generalize/go2ts/pkg/types"
)

type documentRef struct {
	pkgName string
}

var _ go2tstypes.Type = &documentRef{}

func (dr *documentRef) SetPackageName(pkgName string) {
	dr.pkgName = pkgName
}
func (dr *documentRef) GetPackageName() string {
	return dr.pkgName
}
func (dr *documentRef) UsedAsMapKey() bool {
	return false
}
func (dr *documentRef) String() string {
	return "firestore.DocumentRef"
}

func replacer(t types.Type) go2tstypes.Type {
	named, ok := t.(*types.Named)

	if !ok {
		return nil
	}

	if named.String() == "cloud.google.com/go/firestore.DocumentRef" {
		return &documentRef{}
	}

	return nil
}
