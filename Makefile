TEST_OPT=""

.PHONY: test
test: goimports
	go test ./... -v ${TEST_OPT}

.PHONY: statik
statik:
	statik -src ./templates
	gofmt -w ./statik/statik.go

.PHONY: goimports
goimports:
	go get golang.org/x/tools/cmd/goimports
