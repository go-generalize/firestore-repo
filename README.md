# firestore-repo

Cloud firestoreで利用されるコードを自動生成する

# Installation
```console
$ go get github.com/go-generalize/firestore-repo
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
`//go:generate` から始まる行を書くことでfirestore向けのモデルを自動生成するようになる。

また、structの中で一つの要素は必ず `firestore_key:""` を持った要素が必要となっている。  
この要素の型は `string` である必要がある。

## Unique Constraint
`unique` というタグがあるとUniqueという別コレクションにユニーク制約用のドキュメントが生成される。  
電話番号やメールアドレスなど重複を許容したくない場合に使用する。  
この要素の型は `string` である必要がある。

## Various Searches
`Indexes`(map[string]bool型) というフィールドがあると _**[xim](https://github.com/go-utils/xim)**_ を使用したn-gram検索ができるようになる  
対応している検索は、接頭辞/接尾辞/部分一致/完全一致(タグ: prefix/suffix/like/equal)  
_**xim**_ ではUnigram/Bigramしか採用していないため、ノイズが発生しやすい(例: 東京都が京都で検索するとヒットするなど)

この状態で`go generate` を実行すると`_gen.go`で終わるファイルにモデルが生成される。
```commandline
$ go generate
```

### Search Query
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
- Under the [MIT](./LICENSE) License
- Copyright (C) 2020 go-generalize