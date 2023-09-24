# Go GraphQL Armor 🛡️

This is repository inspired by the great work of the [original JS GraphQL Armor](https://github.com/Escape-Technologies/graphql-armor) middleware.
It is dead-simple yet highly customizable security sidecar compatible with any HTTP GraphQL Server or Gateway.

[![Go](https://github.com/ldebruijn/go-graphql-armor/actions/workflows/go.yml/badge.svg)](https://github.com/ldebruijn/go-graphql-armor/actions/workflows/go.yml)

<!-- TOC -->

## Features

* Persisted Operations
* Field Suggestions Redaction

## Installation

Build & Test
```make
    make build
    make test
```

Run Container
```make
    make run_container
```

## Documentation

[Documentation](docs/README.md)

## Configuration

We recommend configuring the binary using a yaml file, place a file called `armor.yml` in the same directory as you're running the binary.

For all the configuration options check out the [Configuration Documentation](docs/configuration.md)

```yaml
web:
  read_timeout: 5s
  write_timeout: 10s
  idle_timeout: 120s
  shutdown_timeout: 20s
  host: 0.0.0.0:8080
  path: /graphql

target:
  host: http://localhost:8081
  timeout: 10s
  keep_alive: 180s

persisted_operations:
  enabled: true
  fail_unknown_operations: true
  store:
    # Only one store will be used
    # Armor will look at all files in the dir and try to load persisted operations from any `.json` file
    dir: "./my-dir"
    # Armor will look at all objects in the bucket and try to load persisted operations from any `.json` file
    gcp_bucket: "gs://somebucket"

field_suggestions:
  enabled: true
```

Alternatively go-graphql-armor can be configured using environment variables or command line arguments.

## Contributing

Ensure you have read the [Contributing Guide](https://github.com/ldebruijn/go-graphql-armor/blob/main/CONTRIBUTING.md) before contributing.

To set up your project, make sure you run the `make dev.setup` script.

```bash
git clone git@github.com:ldebruijn/go-graphql-armor.git
cd go-graphql-armor
make dev.setup
```