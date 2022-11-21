go-generate:
	buf generate --path "api/sys-exporter"
	go generate ./...

test:
	go test -tags "macos_arm_64" -v -count=1 -race -timeout=1m ./...

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.41.1

lint: install-lint-deps
	golangci-lint run ./...

###### EXPORTER ########

build-all-exporter: build-darwin-arm64 build-ubuntu-amd64 build-windows-amd64

build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -tags "macos_arm_64" -o bin/exporter/exporter_arm64_macos cmd/exporter/main.go

build-ubuntu-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "ubuntu_amd_64" -o bin/exporter/exporter_amd64_ubuntu cmd/exporter/main.go

build-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags "windows_amd_64" -o bin/exporter/exporter_amd64_windows.exe cmd/exporter/main.go

###### CLIENT ########

build-all-client: build-client-darwin-arm64 build-client-ubuntu-amd64 build-client-windows-amd64

build-client-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o bin/client/client_arm64_macos cmd/simple_client/main.go

build-client-ubuntu-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/client/client_amd64_ubuntu cmd/simple_client/main.go

build-client-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/client/client_amd64_windows.exe cmd/simple_client/main.go

##### ALL #######

build-all: build-all-client build-all-exporter