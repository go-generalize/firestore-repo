package generator

import (
	"go/types"
	"reflect"
	"strings"

	go2tstypes "github.com/go-generalize/go2ts/pkg/types"
)

const (
	typeString     = "string"
	typeInt        = "int"
	typeInt64      = "int64"
	typeFloat64    = "float64"
	typeBool       = "bool"
	typeTime       = "time.Time"
	typeTimePtr    = "*time.Time"
	typeLatLng     = "latlng.LatLng"
	typeReference  = "firestore.DocumentRef"
	typeMap        = "map[string]"
	typeStringMap  = "map[string]string"
	typeIntMap     = "map[string]int"
	typeInt64Map   = "map[string]int64"
	typeFloat64Map = "map[string]float64"
	typeBoolMap    = "map[string]bool"
)

var (
	supportedTypes = func() []string {
		t := []string{
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

		for i := range t {
			t = append(t, "[]"+t[i])
		}

		return t
	}()
)

func getGoTypeFromGo2ts(t go2tstypes.Type) string {
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
		r := getGoTypeFromGo2ts(t.Inner)

		if strings.HasPrefix(r, "[]") {
			return r
		}

		return "*" + r
	case *go2tstypes.Array:
		return "[]" + getGoTypeFromGo2ts(t.Inner)
	case *go2tstypes.Date:
		return "time.Time"
	case *go2tstypes.Object:
		return ""
	case *go2tstypes.Map:
		return "map[" + getGoTypeFromGo2ts(t.Key) + "]" + getGoTypeFromGo2ts(t.Value)
	case *documentRef:
		return typeReference
	case *latLng:
		return typeLatLng
	}

	panic("unsupported: " + reflect.TypeOf(t).String())
}
