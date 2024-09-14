# Configuration

`graphql-protect` can be configured in various ways, though we recommend configuring it via a `protect.yml`. file

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
  target:
    redacted_headers: []
      
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

# Configures if upstream errors need to be obfuscated, this can help you hide internals of your upstream landscape
obfuscate_upstream_errors: true
    
persisted_operations:
  # Enable or disable the feature, disabled by default
  enabled: false
  # Fail unknown operations, disable this feature to allow unknown operations to reach your GraphQL API
  reject_on_failure: true
  # Loader decides how persisted operations are loaded, see loader chapter for more details
  loader:
    # Type of loader to use
    type: local
    # Location to load persisted operations from
    location: ./store
    # Whether to reload persisted operations periodically
    reload:
      enabled: true
      # The interval in which the persisted operations are refreshed
      interval: 5m0s
      # The timeout for the refreshing operation
      timeout: 10s

block_field_suggestions:
  enabled: true
  mask: "[redacted]"

max_aliases:
  # Enable the feature
  enabled: true
  # The maximum number of allowed aliases within a single request.
  max: 15
  # Reject the request when the rule fails. Disable this to allow the request
  reject_on_failure: true

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
  # Enable enforcing POST http method
  enabled: true

# Enable or disable logging of graphql errors
log_graphql_errors: false

log:
  # text, or json for structured logging
  format: text
```

For a more in-depth view of each option visit the accompanying documentation page of each individual protection.

## Graphql protect - validate run mode
While the validate run mode works with the same config as the normal mode, for simplicity's sake you can leave out quite some unused options.
As an example checkout the config below:

```yaml
schema:
# Path to a local file in which the schema can be found
path: "./schema.graphql"

persisted_operations:
    enabled: true
    # Store is the location on local disk where graphql-protect can find the persisted operations, it loads any `*.json` files on disk
    loader:
      # Type of loader to use
      type: local
      # Location to load persisted operations from
      location: ./store

max_aliases:
    # Enable the feature
    enabled: true
    # The maximum number of allowed aliases within a single request.
    max: 15

block_field_suggestions:
    enabled: true
    mask: "[redacted]"

max_depth:
    enabled: true
    # The maximum allowed depth within a single request.
    max: 15

max_tokens:
    # Enable the feature
    enabled: true
    # The maximum number of allowed tokens within a single request.
    max: 10000

max_batch:
    # Enable the feature
    enabled: true
    # The maximum number of operations within a single batched request.
    max: 5
```
