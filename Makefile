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
	cd /tmp && go get golang.org/x/tools/cmd/goimports

.PHONY: code_clean
code_clean:
	cd testfiles && rm -rf */*_gen.go
