// Code generated by firestore-repo. DO NOT EDIT.
// generated version: 1.12.0
package repository

import "golang.org/x/xerrors"

// OpType - operator type
type OpType = string

const (
	OpTypeEqual              OpType = "=="
	OpTypeNotEqual           OpType = "!="
	OpTypeLessThan           OpType = "<"
	OpTypeLessThanOrEqual    OpType = "<="
	OpTypeGreaterThan        OpType = ">"
	OpTypeGreaterThanOrEqual OpType = ">="
	OpTypeIn                 OpType = "in"
	OpTypeNotIn              OpType = "not-in"
	OpTypeArrayContains      OpType = "array-contains"
	OpTypeArrayContainsAny   OpType = "array-contains-any"
)

// FilterType - extra indexes filters type
type FilterType = int

const (
	FilterTypeAdd FilterType = 1 << iota
	FilterTypeAddPrefix
	FilterTypeAddSuffix
	FilterTypeAddBiunigrams
	FilterTypeAddSomething
)

var (
	ErrAlreadyExists        = xerrors.New("already exists")
	ErrAlreadyDeleted       = xerrors.New("already been deleted")
	ErrNotFound             = xerrors.New("not found")
	ErrLogicallyDeletedData = xerrors.New("logically deleted data")
	ErrUniqueConstraint     = xerrors.New("unique constraint error")
	ErrNotAvailableCG       = xerrors.New("not available in collection groups")
	ErrVersionConflict      = xerrors.New("version conflict")
)

type (
	UniqueMiddlewareKey      struct{}
	transactionInProgressKey struct{}
)
