package examples

//go:generate firestore-repo -disable-meta -sub-collection History

// History - Task sub-collection
type History struct {
	ID              string `firestore:"-" firestore_key:"auto"`
	IsSubCollection bool   ``
}
