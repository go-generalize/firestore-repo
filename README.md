# firestore-repo

Automatically generate code used by Cloud Firestore.  

[日本語ドキュメント](./docs/ja.md)

# Installation
Recommend that you drop the binary from the release and use it.  
Also, possible with `go install` ↓
```console
$ go install github.com/go-generalize/firestore-repo
```

# Usage

```go
package task

import (
	"time"
)

//go:generate firestore-repo -disable-meta Task

type Task struct {
	ID      string          `firestore:"-"           firestore_key:""`
	Desc    string          `firestore:"description" indexer:"suffix,like" unique:""`
	Done    bool            `firestore:"done"        indexer:"equal"`
	Created time.Time       `firestore:"created"`
	Indexes map[string]bool `firestore:"indexes"`
}
```
By writing a line starting with `go:generate`, the model for firestore will be automatically generated.  

When used in SubCollection, add the argument `-sub-collection`.  

If you want to use Meta information (such as CreatedAt and Version used in optimistic exclusive lock) together,  
you can use it by removing the argument `-disable-meta` and embedding a structure whose suffix is Meta.  

The format of the Meta structure is as follows.
```go
type Meta struct {
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	UpdatedBy string
	DeletedAt *time.Time
	DeletedBy string
	Version   int
}
```

Also, one element in the struct must have an element with `firestore_key:""`.  
The type of this element must be `string`.  
ID is automatically generated by setting `firestore_key:"auto"`.  

If you execute `go generate` in this state, the model will be generated in a file with the suffix` _gen.go`.  
```commandline
$ go generate
```

## Unique constraint
If there is a tag called `unique`, a document for unique constraints will be generated in another collection called Unique.  
Use when you do not want to allow duplicates such as phone numbers and email addresses.  
The type of this element must be `string`.

## Search diversity
The existence of a field called `Indexes` (`map[string]bool` type) enables n-gram search using _**[xim](https://github.com/go-utils/xim)**_.  
Corresponding search is prefix/suffix/partial/exact match. (tag: prefix/suffix/like/equal)  
Since _**xim**_ uses only Unigram Bigram, it is prone to noise (eg, when `東京都` searches in `京都`, it hits).

### Search query
Task.Desc = "Hello, World!".
- Partial match search
```go
req := &model.TaskListReq{
	Desc: model.NewQueryChainer().Filters("o, Wor", model.FilterTypeAddBiunigrams),
}

tasks, err := taskRepo.List(ctx, req, nil)
if err != nil {
	// error handling
}
```

- Prefix match search
```go
req := &model.TaskListReq{
	Desc: model.NewQueryChainer().Filters("Hell", model.FilterTypeAddPrefix),
}

tasks, err := taskRepo.List(ctx, req, nil)
if err != nil {
	// error handling
}
```

- Suffix match search
```go
req := &model.TaskListReq{
	Desc: model.NewQueryChainer().Filters("orld!", model.FilterTypeAddSuffix),
}

tasks, err := taskRepo.List(ctx, req, nil)
if err != nil {
	// error handling
}
```

- Exact match search
```go
chainer := model.NewQueryChainer
req := &model.TaskListReq{
	Desc: chainer().Filters("Hello, World!", model.FilterTypeAdd),
	Done: chainer().Filters(true, model.FilterTypeAddSomething), // Use Add Something when it is not a string.
}

tasks, err := taskRepo.List(ctx, req, nil)
if err != nil {
	// error handling
}
```

## License
- Under the [MIT License](./LICENSE)
- Copyright (C) 2021 go-generalize
