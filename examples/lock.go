package examples

//go:generate firestore-repo -o different_dir Lock

// Lock - with automatic id generation
type Lock struct {
	ID   string             `firestore:"-" firestore_key:"auto"`
	Text string             `firestore:"text" unique:""`
	Flag map[string]float64 `firestore:"flag"`
	Meta
}
