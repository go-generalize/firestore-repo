package model

import "github.com/go-generalize/firestore-repo/generator/testfiles/auto/meta"

//go:generate firestore-repo LockMeta2

// Lock ID自動生成あり
type LockMeta2 struct {
	ID   string             `firestore:"-" firestore_key:"auto"`
	Text string             `firestore:"text"`
	Flag map[string]float64 `firestore:"flag"`
	meta.AAAMeta
}
