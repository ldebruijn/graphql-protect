# GraphQL Protect 🛡️

GraphQL Protect is dead-simple yet highly customizable security proxy compatible with any HTTP GraphQL Server or Gateway.

![GraphQL Protect Banner](docs/assets/banner.jpeg?raw=true)

[![Go](https://github.com/ldebruijn/graphql-protect/actions/workflows/go.yml/badge.svg)](https://github.com/ldebruijn/graphql-protect/actions/workflows/go.yml)

_This is repository inspired by the great work of the Javascript [GraphQL Armor](https://github.com/Escape-Technologies/graphql-armor) middleware._

<!-- TOC -->

## Features

* [Persisted Operations](docs/protections/persisted_operations.md)
* [Block Field Suggestions](docs/protections/block_field_suggestions.md)
* [Obfuscate upstream errors](docs/protections/obfuscate_upstream_errors.md)
* [Max Aliases](docs/protections/max_aliases.md)
* [Max Tokens](docs/protections/max_tokens.md)
* [Max Depth](docs/protections/max_depth.md)
* [Max Batch](docs/protections/max_batch.md)
* [Enforce POST](docs/protections/enforce_post.md)
* _Max Directives (coming soon)_
* _Cost Limit (coming soon)_


Curious why you need these features? Check out this [Excellent talk on GraphQL security](https://www.youtube.com/watch?v=hyB2UKsEkqA&list=PLP1igyLx8foE9SlDLI1Vtlshcon5r1jMJ) on YouTube.

## Nice to know
Currently, handling response encoding from the upstream is not supported, we're open for contributions : )


## Installation

### As Container
```shell
docker pull ghcr.io/ldebruijn/graphql-protect:latest
docker run -p 8080:8080 -v $(pwd)/protect.yml:/app/protect.yml -v $(pwd)/schema.graphql:/app/schema.graphql ghcr.io/ldebruijn/graphql-protect:latest
```
Make sure to portforward the right ports for your supplied configuration

Check out our [run documentation](docs/README.md#run) for more concrete examples.

### Source code

```shell
git clone git@github.com:ldebruijn/graphql-protect.git
```

Build & Test
```shell
    make build
    make test
```

Run Container
```shell
    make run_container
```

## Documentation

Check out our extensive documentation, including configuration examples, detailed descriptions of each protection feature as well as deployment configuration examples.

[Documentation](docs/README.md)

## Configuration

We recommend configuring the binary using a yaml file, place a file called `protect.yml` in the same directory as you're running the binary.

For all the configuration options check out the [Configuration Documentation](docs/configuration.md)

Alternatively graphql-protect can be configured using environment variables or command line arguments.

## Contributing

Ensure you have read the [Contributing Guide](https://github.com/ldebruijn/graphql-protect/blob/main/CONTRIBUTING.md) before contributing.

To set up your project, make sure you run the `make dev.setup` script.

```bash
git clone git@github.com:ldebruijn/graphql-protect.git
cd graphql-protect
make dev.setup
```
