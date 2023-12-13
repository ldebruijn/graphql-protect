# Configuration

go-graphql-armor can be configured in various ways, though we recommend configuring it via a `armor.yml`. file

<!-- TOC -->

# armor.yml

The best way to configure `go-graphql-armor` is by specifying a `armor.yml` in the same directory as you're running the binary.

The following outlines the structure of the yaml

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
  # Enable or disable the feature, enabled by default
  enabled: "true"
  # Fail unknown operations, disable this feature to allow unknown operations to reach your GraphQL API
  fail_unknown_operations: "true"
  # Store is the location on local disk where go-graphql-armor can find the persisted operations, it loads any `*.json` files on disk
  store: "./store"
  reload:
    enabled: "true"
    # The interval in which the local store dir is read and refreshes the internal state
    interval: 5m
    # The timeout for the remote operation
    timeout: 10s
  remote:
    # Load persisted operations from a GCP Cloud Storage bucket.
    # Will look at all the objects in the bucket and try to load any object with a `.json` extension
    gcp_bucket: "gs://somebucket"

max_aliases:
  # Enable the feature
  enable: "true"
  # The maximum number of allowed aliases within a single request.
  max: 15
  # Reject the request when the rule fails. Disable this to allow the request
  reject_on_failure: "true"

block_field_suggestions:
  enabled: "true"
  mask: [redacted]
```

For a more in-depth view of each option visit the accompanying documentation page.

## Environment Variables

If so desired `go-graphql-armor` _can_ be configured using environment variables. write out the full configuration path for each value.

For example:

```bash
PERSISTED_OPERATIONS_ENABLED: true
WEB_PATH: /graphql
PERSISTED_OPERATIONS_STORE_GCP_BUCKET: gs://my-bucket
```

## Command line arguments

Usage: go-graphql-armor [options] [arguments]

Examples:

```bash
go-graphql-armor \
    --persisted-operations-enabled=true \
    --web-path=/graphql \
    --persisted-operations-store-gcp-bucket=gs://my-bucket
```

## Which configuration is applied?

During startup `go-graphql-armor` will output its applied configuration. It will do this in command line argument format, though it will apply and output configuration from any of these sources.