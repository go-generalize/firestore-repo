package model

//go:generate firestore-repo -s SubTask

type SubTask struct {
	ID              string `firestore:"-" firestore_key:"auto"`
	IsSubCollection bool   ``
}
