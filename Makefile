TEST_OPT=""

.PHONY: bootstrap
bootstrap:
	mkdir -p bin
	GOBIN=$(PWD)/bin go install github.com/golang/mock/mockgen@latest

.PHONY: test
test: goimports
	go test ./... -v ${TEST_OPT}

.PHONY: goimports
goimports:
	cd /tmp && go get golang.org/x/tools/cmd/goimports

.PHONY: code_clean
code_clean:
	cd testfiles && rm -rf */*_gen.go

.PHONY: lint
lint:
	golangci-lint run --config ".github/.golangci.yml" --fast

.PHONY: build
build:
	go build -o ./bin/firestore-repo .
