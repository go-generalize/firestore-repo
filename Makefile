TEST_OPT=""

.PHONY: test
test:
	go test ./... -v ${TEST_OPT}

.PHONY: statik
statik:
	statik -src ./templates
	gofmt -w ./statik/statik.go