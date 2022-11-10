go-generate:
	buf generate --path "api/sys-exporter"
	go generate ./...

test:
	go test -v -count=1 -race -timeout=1m ./...

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.41.1

lint: install-lint-deps
	golangci-lint run ./...
