package model

//go:generate firestore-repo LockMeta
//go:generate firestore-repo -o different_dir LockMeta

// Lock ID自動生成あり
type LockMeta struct {
	ID   string             `firestore:"-" firestore_key:"auto"`
	Text string             `firestore:"text"`
	Flag map[string]float64 `firestore:"flag"`
	Meta
}
