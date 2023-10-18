# Go GraphQL Armor 🛡️

This is repository inspired by the great work of the [original JS GraphQL Armor](https://github.com/Escape-Technologies/graphql-armor) middleware.
It is dead-simple yet highly customizable security sidecar compatible with any HTTP GraphQL Server or Gateway.

[![Go](https://github.com/ldebruijn/go-graphql-armor/actions/workflows/go.yml/badge.svg)](https://github.com/ldebruijn/go-graphql-armor/actions/workflows/go.yml)

<!-- TOC -->

## Features

* [Persisted Operations](docs/persisted_operations.md)
* [Block Field Suggestions](docs/block_field_suggestions.md)
* _Max Aliases (coming soon)_
* _Max Depth (coming soon)_
* _Max Directives (coming soon)_
* _Max Tokens (coming soon)_
* _Cost Limit (coming soon)_

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

Alternatively go-graphql-armor can be configured using environment variables or command line arguments.

## Contributing

Ensure you have read the [Contributing Guide](https://github.com/ldebruijn/go-graphql-armor/blob/main/CONTRIBUTING.md) before contributing.

To set up your project, make sure you run the `make dev.setup` script.

```bash
git clone git@github.com:ldebruijn/go-graphql-armor.git
cd go-graphql-armor
make dev.setup
```