package model

import (
	"time"
)

//go:generate firestore-repo Task

// Task ID自動生成あり
type Task struct {
	ID         string             `firestore:"-" firestore_key:"auto"`
	Desc       string             `firestore:"description"`
	Created    time.Time          `firestore:"created"`
	Done       bool               `firestore:"done"`
	Done2      bool               `firestore:"done2"`
	Count      int                `firestore:"count"`
	Count64    int64              `firestore:"count64" op:"<="`
	NameList   []string           `firestore:"nameList"`
	Proportion float64            `firestore:"proportion"`
	Flag       map[string]float64 `firestore:"flag"`
}
