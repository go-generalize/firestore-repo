package main

const (
	biunigrams     = "Biunigrams"
	prefix         = "Prefix"
	queryLabel     = "QueryLabel"
	typeString     = "string"
	typeInt        = "int"
	typeInt64      = "int64"
	typeFloat64    = "float64"
	typeBool       = "bool"
	typeTime       = "time.Time"
	typeLatLng     = "*latlng.LatLng"
	typeMap        = "map[string]"
	typeStringMap  = "map[string]string"
	typeIntMap     = "map[string]int"
	typeInt64Map   = "map[string]int64"
	typeFloat64Map = "map[string]float64"
	NgType         = "NG"
)

type Operator string

const (
	OperatorLessThan           Operator = "<"
	OperatorLessThanOrEqual    Operator = "<="
	OperatorGreaterThan        Operator = ">"
	OperatorGreaterThanOrEqual Operator = ">="
	OperatorEqual              Operator = "=="
	OperatorIn                 Operator = "in"
	OperatorArrayContains      Operator = "array-contains"
	OperatorArrayContainsAny   Operator = "array-contains-any"
)
