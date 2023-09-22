
.PHONY: dev.setup
dev.setup:
	go mod tidy

.PHONY: build
build:
	# build flags
	go build ./...

.PHONY: run_container
build_container:
	docker build github.com/ldebruijn/go-graphql-armor -t go-graphql-armor .

.PHONY: run_container
run_container:
	go run -d -p 8080:8080 go-graphql-armor