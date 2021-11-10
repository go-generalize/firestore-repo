package main

import (
	"go/types"

	go2tstypes "github.com/go-generalize/go2ts/pkg/types"
)

type documentRef struct {
	go2tstypes.Common
}

var _ go2tstypes.Type = &documentRef{}

// UsedAsMapKey returns whether this type can be used as the key for map
func (dr *documentRef) UsedAsMapKey() bool {
	return false
}

// String returns this type in string representation
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
