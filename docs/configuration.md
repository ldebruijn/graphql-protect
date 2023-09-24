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