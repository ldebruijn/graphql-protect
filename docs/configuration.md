# Configuration

graphql-protect can be configured in various ways, though we recommend configuring it via a `protect.yml`. file

<!-- TOC -->

# protect.yml

The best way to configure `graphql-protect` is by specifying a `protect.yml` in the same directory as you're running the binary.

The following outlines the structure of the yaml, as well as outlines the **defaults** for each configuration option.

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

schema:
  # Path to a local file in which the schema can be found
  path: "./schema.graphql"
  # Automatically reload the schema file. 
  # It will reload the contents of the file referenced by the `schema.path` configuration option
  # after each `schema.auto_reload.interval` has passed.
  auto_reload:
    # Enable automatic file reloading
    enabled: true
    # The interval in which the schema file should be reloaded
    interval: 5m
    
# Configures whether we obfuscate graphql-protect validation errors such as max_aliases/max_tokens
# Recommended to set it to 'true' for public environments
obfuscate_validation_errors: false    
    
persisted_operations:
  # Enable or disable the feature, enabled by default
  enabled: false
  # Fail unknown operations, disable this feature to allow unknown operations to reach your GraphQL API
  reject_on_failure: true
  # Store is the location on local disk where graphql-protect can find the persisted operations, it loads any `*.json` files on disk
  store: "./store"
  reload:
    enabled: true
    # The interval in which the local store dir is read and refreshes the internal state
    interval: 5m
    # The timeout for the remote operation
    timeout: 10s
  remote:
    # Load persisted operations from a GCP Cloud Storage bucket.
    # Will look at all the objects in the bucket and try to load any object with a `.json` extension
    gcp_bucket: ""

max_aliases:
  # Enable the feature
  enabled: true
  # The maximum number of allowed aliases within a single request.
  max: 15
  # Reject the request when the rule fails. Disable this to allow the request
  reject_on_failure: true

block_field_suggestions:
  enabled: true
  mask: [redacted]
  
max_depth:
  enabled: true
  # The maximum allowed depth within a single request.
  max: 15
  # Reject the request when the rule fails. Disable this to allow the request
  reject_on_failure: true

max_tokens:
  # Enable the feature
  enabled: true
  # The maximum number of allowed tokens within a single request.
  max: 10000
  # Reject the request when the rule fails. Disable this to allow the request regardless of token count.
  reject_on_failure: true

max_batch:
  # Enable the feature
  enabled: true
  # The maximum number of operations within a single batched request.
  max: 5
  # Reject the request when the rule fails. Disable this to allow the request regardless of token count.
  reject_on_failure: true

enforce_post:
  # Enable the feature
  enabled: true
```

For a more in-depth view of each option visit the accompanying documentation page of each individual protection.

## Environment Variables

If so desired `graphql-protect` _can_ be configured using environment variables. write out the full configuration path for each value.

For example:

```bash
PERSISTED_OPERATIONS_ENABLED: true
WEB_PATH: /graphql
PERSISTED_OPERATIONS_STORE_GCP_BUCKET: gs://my-bucket
```

## Command line arguments

Usage: `graphql-protect [options] [arguments]`

Examples:

```bash
graphql-protect \
    --persisted-operations-enabled=true \
    --web-path=/graphql \
    --persisted-operations-store-gcp-bucket=gs://my-bucket
```

## Which configuration is applied?

During startup `graphql-protect` will output its applied configuration. It will do this in command line argument format, though it will apply and output configuration from any of these sources.