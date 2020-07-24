package model

//go:generate firestore-repo -disable-meta -sub-collection SubTask

type SubTask struct {
	ID              string `firestore:"-" firestore_key:"auto"`
	IsSubCollection bool   ``
}
