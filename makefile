
SHORT_HASH = $(shell git rev-parse --short HEAD)

META_PKG = main
LDFLAGS += -X '$(META_PKG).build=$(SHORT_HASH)'
LDFLAGS += -s -w

.PHONY: dev.setup
dev.setup:
	go mod tidy

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" ./cmd/main.go

.PHONY: test
test:
	go test -v ./...

.PHONY: lint
## Runs a linter over the code
lint:
	golangci-lint run --timeout 3m

.PHONY: run_container
build_container: build
	docker build github.com/ldebruijn/go-graphql-armor -t go-graphql-armor .

.PHONY: run_container
run_container: build_container
	go run -d -p 8080:8080 go-graphql-armor