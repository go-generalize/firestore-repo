# firestore-repo

Cloud firestoreで利用されるコードを自動生成する

### Installation
```console
$ go get github.com/go-generalize/firestore-repo
```

### Usage

```go
package task

import (
	"time"
)

//go:generate firestore-repo Task

type Task struct {
	ID      string         `firestore:"-" firestore_key:""`
	Desc    string         `firestore:"description"`
	Done    bool           `firestore:"done"`
	Created time.Time      `firestore:"created"`
}
```
`//go:generate` から始まる行を書くことでfirestore向けのモデルを自動生成するようになる。

また、structの中で一つの要素は必ず`firestore_key:""`を持った要素が必要となっている。  
この要素の型は `string`である必要がある。

この状態で`go generate` を実行すると`_gen.go`で終わるファイルにモデルが生成される。
```commandline
$ go generate
```

#### Search Query
Task.Desc = "Hello, World!".
- Bigrams / Unigrams
```go
req := &model.TaskListReq{
	Desc: model.NewQueryChainer().Filters("o, Wor", model.FilterTypeAddBiunigrams),
}

tasks, err := taskRepo.List(ctx, req, nil)
if err != nil {
	// error handling
}
```

- Prefix
```go
req := &model.TaskListReq{
	Desc: model.NewQueryChainer().Filters("Hell", model.FilterTypeAddPrefix),
}

tasks, err := taskRepo.List(ctx, req, nil)
if err != nil {
	// error handling
}
```

- Suffix
```go
req := &model.TaskListReq{
	Desc: model.NewQueryChainer().Filters("orld!", model.FilterTypeAddSuffix),
}

tasks, err := taskRepo.List(ctx, req, nil)
if err != nil {
	// error handling
}
```

## License
- Under the MIT License
- Copyright (C) 2020 go-generalize