# Go GraphQL Armor 🛡️

This is repository inspired by the great work of the ![original JS GraphQL Armor]https://github.com/Escape-Technologies/graphql-armor
It is dead-simple yet highly customizable security sidecar compatible with any HTTP GraphQL Server or Gateway.

![GraphQL-Armor banner](https://raw.githubusercontent.com/Escape-Technologies/graphql-armor/main/services/docs/static/img/banner.png)

[![CI](https://github.com/Escape-Technologies/graphql-armor/actions/workflows/ci.yaml/badge.svg)](https://github.com/Escape-Technologies/graphql-armor/actions/workflows/ci.yaml) [![release](https://github.com/Escape-Technologies/graphql-armor/actions/workflows/release.yaml/badge.svg)](https://github.com/Escape-Technologies/graphql-armor/actions/workflows/release.yaml) [![e2e](https://github.com/Escape-Technologies/graphql-armor/actions/workflows/e2e.yaml/badge.svg)](https://github.com/Escape-Technologies/graphql-armor/actions/workflows/e2e.yaml) ![npm](https://img.shields.io/npm/v/@escape.tech/graphql-armor) [![codecov](https://codecov.io/gh/Escape-Technologies/graphql-armor/branch/main/graph/badge.svg)](https://codecov.io/gh/Escape-Technologies/graphql-armor)

## Installation

```makefile
    make build
```

```makefile
    make build_container
    make run_container
```

## Documentation

[//]: # ( github pages)
[https://escape.tech/graphql-armor/docs/getting-started](https://escape.tech/graphql-armor/docs/getting-started)

## Configuration

We recommend configuring the binary using a yaml file, place a file called `armor.yml` in the same directory as you're running the binary.

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
  allow_unpersisted_operations: false
  fail_unknown_operations: true
  store:
    # Only one store will be used
    # Armor will look at all files in the dir and try to load persisted operations from any `.json` file
    dir: "./my-dir"
    # Armor will look at all objects in the bucket and try to load persisted operations from any `.json` file
    gcp_bucket: "gs://somebucket"
```

Alternatively go-graphql-armor can be configured using environment variables or command line arguments.

## Contributing

Ensure you have read the [Contributing Guide](https://github.com/Escape-Technologies/graphql-armor/blob/main/CONTRIBUTING.md) before contributing.

To set up your project, make sure you run the `make dev.setup` script.

```bash
git clone git@github.com:ldebruijn/go-graphql-armor.git
cd go-graphql-armor
make dev.setup
```

```http request
curl -v -X POST -d "{ \"extensions\": { \"persistedQuery\": { \"sha256Hash\": \"foobar\"}}  }" -H "content-type: application/json" http://localhost:8080/graphql
```