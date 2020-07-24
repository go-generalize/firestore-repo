package model

import "time"

//go:generate firestore-repo Lock

type Meta struct {
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	UpdatedBy string
	DeletedAt *time.Time
	DeletedBy string
	Version   int
}

// Lock ID自動生成あり
type Lock struct {
	ID   string             `firestore:"-" firestore_key:"auto"`
	Text string             `firestore:"text"`
	Flag map[string]float64 `firestore:"flag"`
	Meta
}
