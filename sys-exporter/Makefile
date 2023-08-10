go-generate:
	buf generate --path "api/sys-exporter"
	go generate ./...

test:
	go test -tags "macos" -v -count=1 -race -timeout=1m ./...

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.41.1

lint: install-lint-deps
	golangci-lint run ./...

###### DOCKER-COMPOSE ######

up:
	docker compose up -d

down:
	docker compose down

###### EXPORTER ########

build-all-exporter: build-client-ubuntu-arm64 build-darwin-arm64 build-ubuntu-amd64 build-windows-amd64

build-ubuntu-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags "ubuntu" -o bin/exporter/exporter_arm64_ubuntu cmd/exporter/main.go

build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -tags "macos" -o bin/exporter/exporter_arm64_macos cmd/exporter/main.go

build-ubuntu-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "ubuntu" -o bin/exporter/exporter_amd64_ubuntu cmd/exporter/main.go

build-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags "windows" -o bin/exporter/exporter_amd64_windows.exe cmd/exporter/main.go

###### CLIENT ########

build-all-client: build-client-ubuntu-arm64 build-client-darwin-arm64 build-client-ubuntu-amd64 build-client-windows-amd64

build-client-ubuntu-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/client/client_arm64_ubuntu cmd/simple_client/main.go

build-client-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o bin/client/client_arm64_macos cmd/simple_client/main.go

build-client-ubuntu-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/client/client_amd64_ubuntu cmd/simple_client/main.go

build-client-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/client/client_amd64_windows.exe cmd/simple_client/main.go

###### DOCKER-EXPORTER #######

docker-build-ubuntu-arm64: build-ubuntu-arm64
	docker buildx build --build-arg PATH_TO_BIN="bin/exporter/exporter_arm64_ubuntu" -f build/Dockerfile . -t docker.io/soundsofanarchy/exporter:arm64-latest --push --platform=linux/arm64

###### DOCKER-CLIENT ######

docker-build-client-ubuntu-arm64: build-client-ubuntu-arm64
	docker buildx build --build-arg PATH_TO_BIN="bin/client/client_arm64_ubuntu" -f build/Dockerfile . -t docker.io/soundsofanarchy/exporter-client:arm64-latest --push --platform=linux/arm64

##### ALL #######

build-all: build-all-client build-all-exporter

build-docker-all: docker-build-ubuntu-arm64 docker-build-client-ubuntu-arm64