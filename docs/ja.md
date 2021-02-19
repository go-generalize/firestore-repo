# firestore-repo

Cloud firestoreで利用されるコードを自動生成する。

# インストール
リリースからバイナリを落としてきて使用する事をオススメする。  
go installでも可能↓
```console
$ go install github.com/go-generalize/firestore-repo
```

# 使用方法

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

SubCollectionで利用される場合は `-sub-collection` という引数を追加する。  

Meta情報(CreatedAtや楽観排他ロックで使用するVersionなど)を併用したい場合は  
`-disable-meta` という引数を外し接尾辞がMetaという構造体を埋め込むことで利用できる。
Meta構造体のフォーマットは↓
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

また、structの中で一つの要素は必ず `firestore_key:""` を持った要素が必要となっている。  
この要素の型は `string` である必要がある。  
`firestore_key:"auto"` とすることによりIDが自動生成される。  

この状態で`go generate` を実行すると接尾辞が `_gen.go` のファイルにモデルが生成される。
```commandline
$ go generate
```

## ユニーク制約
`unique` というタグがあるとUniqueという別コレクションにユニーク制約用のドキュメントが生成される。  
電話番号やメールアドレスなど重複を許容したくない場合に使用する。  
この要素の型は `string` である必要がある。

## 検索の多様性
`Indexes`(map[string]bool型) というフィールドがあると _**[xim](https://github.com/go-utils/xim)**_ を使用したn-gram検索ができるようになる  
対応している検索は、接頭辞/接尾辞/部分一致/完全一致(タグ: prefix/suffix/like/equal)  
_**xim**_ ではUnigram/Bigramしか採用していないため、ノイズが発生しやすい(例: 東京都が京都で検索するとヒットするなど)

### 検索クエリ
Task.Desc = "Hello, World!".
- 部分一致検索
```go
req := &model.TaskListReq{
	Desc: model.NewQueryChainer().Filters("o, Wor", model.FilterTypeAddBiunigrams),
}

tasks, err := taskRepo.List(ctx, req, nil)
if err != nil {
	// error handling
}
```

- 接頭辞一致検索
```go
req := &model.TaskListReq{
	Desc: model.NewQueryChainer().Filters("Hell", model.FilterTypeAddPrefix),
}

tasks, err := taskRepo.List(ctx, req, nil)
if err != nil {
	// error handling
}
```

- 接尾辞一致検索
```go
req := &model.TaskListReq{
	Desc: model.NewQueryChainer().Filters("orld!", model.FilterTypeAddSuffix),
}

tasks, err := taskRepo.List(ctx, req, nil)
if err != nil {
	// error handling
}
```

- 完全一致検索
```go
chainer := model.NewQueryChainer
req := &model.TaskListReq{
	Desc: chainer().Filters("Hello, World!", model.FilterTypeAdd),
	Done: chainer().Filters(true, model.FilterTypeAddSomething), // 文字列以外の時はAddSomethingを使用する
}

tasks, err := taskRepo.List(ctx, req, nil)
if err != nil {
	// error handling
}
```

## License
- Under the [MIT License](../LICENSE)
- Copyright (C) 2021 go-generalize
