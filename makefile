
SHORT_HASH = $(shell git rev-parse --short HEAD)

META_PKG = main
LDFLAGS += -X '$(META_PKG).build=$(SHORT_HASH)'
LDFLAGS += -s -w

.PHONY: dev.setup
dev.setup:
	go mod tidy

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" ./cmd/main.go

.PHONY: run_container
build_container:
	docker build github.com/ldebruijn/go-graphql-armor -t go-graphql-armor .

.PHONY: run_container
run_container:
	go run -d -p 8080:8080 go-graphql-armor