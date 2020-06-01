package model

import (
	"time"
)

//go:generate firestore-repo Task

// Task ID自動生成なし
type Task struct {
	Identity   string    `firestore:"-" firestore_key:""`
	Desc       string    `firestore:"description"`
	Created    time.Time `firestore:"created"`
	Done       bool      `firestore:"done"`
	Done2      bool      `firestore:"done2"`
	Count      int       `firestore:"count"`
	Count64    int64     `firestore:"count64" op:">="`
	NameList   []string  `firestore:"nameList"`
	Proportion float64   `firestore:"proportion"`
	Flag       Flag      `firestore:"flag"`
}

type Flag bool
