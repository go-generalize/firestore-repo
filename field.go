package main

type Field struct {
	Name       string
	Type       string
	ParentPath string
	IsEmbed    bool
	IsPointer  bool
}

type MetaField struct {
	Require          bool
	RequireType      string
	Find             bool
	FindType         string
	RequireIsPointer bool
	FindIsPointer    bool
}
