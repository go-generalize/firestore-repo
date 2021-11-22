package generator

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

type latLng struct {
	go2tstypes.Common
}

var _ go2tstypes.Type = &latLng{}

// UsedAsMapKey returns whether this type can be used as the key for map
func (dr *latLng) UsedAsMapKey() bool {
	return false
}

// String returns this type in string representation
func (dr *latLng) String() string {
	return "latlng.LatLng"
}

type unsupportedFunction struct {
	go2tstypes.Common
}

var _ go2tstypes.Type = &unsupportedFunction{}

// UsedAsMapKey returns whether this type can be used as the key for map
func (uf *unsupportedFunction) UsedAsMapKey() bool {
	return false
}

// String returns this type in string representation
func (uf *unsupportedFunction) String() string {
	return "<unsupported function type>"
}

// replacer replaces types for firestore with specific types in go2tsparser
func replacer(t types.Type) go2tstypes.Type {
	_, ok := t.(*types.Signature)

	if ok {
		return &unsupportedFunction{}
	}

	named, ok := t.(*types.Named)

	if !ok {
		return nil
	}

	if named.String() == "cloud.google.com/go/firestore.DocumentRef" {
		return &documentRef{}
	}

	if named.String() == "google.golang.org/genproto/googleapis/type/latlng.LatLng" {
		return &latLng{}
	}

	return nil
}
