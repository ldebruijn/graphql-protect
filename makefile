
SHORT_HASH = $(shell git rev-parse --short HEAD)
BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
VERSION = develop

META_PKG = main
LDFLAGS += -X '$(META_PKG).shortHash=$(SHORT_HASH)'
LDFLAGS += -X '$(META_PKG).build=$(VERSION)'
LDFLAGS += -s -w

.PHONY: dev.setup
dev.setup:
	go mod tidy
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.1

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o main ./cmd/.

x-build:
	GOOS=linux GOARCH=amd64 make build

.PHONY: test
test:
	go test -v ./...

.PHONY: lint
## Runs a linter over the code
lint:
	golangci-lint run --timeout 3m

.PHONY: build_container
build_container: build
	docker build . -t graphql-protect --build-arg BUILD_DATE=$(BUILD_DATE) --build-arg VERSION=$(VERSION) --build-arg REVISION=$(SHORT_HASH)

x_build_container: x-build
	docker buildx build --platform="linux/amd64" -t graphql-protect --build-arg BUILD_DATE=$(BUILD_DATE) --build-arg VERSION=$(VERSION) --build-arg REVISION=$(SHORT_HASH) .

.PHONY: run_container
run_container: build_container
	docker run -itd --rm -p 8080:8080 graphql-protect