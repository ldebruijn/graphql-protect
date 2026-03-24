# Access Logging

In some cases you want to keep a record of what operations were performed against your landscape. The access logging protection can provide that for you.

Access logging supports two modes:
- **Stdout Mode (Default)**: All logs (access logs + application logs) are written to stdout
- **Google Cloud Logging Mode**: Access logs are sent to Google Cloud Logging, while application logs remain on stdout

<!-- TOC -->

## Logging Modes

### Stdout Mode (Default)

When Google Cloud Logging is disabled (the default), all access logs are written to stdout using structured logging (slog). This mode supports async buffering for high-throughput scenarios.

### Google Cloud Logging Mode

When Google Cloud Logging is enabled, access logs are sent to Google Cloud Logging, while application logs (startup messages, errors, etc.) remain on stdout. The GCP client handles batching natively, so async mode is automatically disabled in this configuration (even if set to true in the config).

**Important**: In GCP mode, only access logs go to Google Cloud Logging. All other application logs continue to be written to stdout for debugging and operational visibility.

## Configuration

### Basic Configuration (Stdout Mode)

You can configure `graphql-protect` to enable access logging for incoming operaitons.

```yaml
access_logging:
  # Enable the feature, 
  enabled: true
  include_headers:
    # Include any headers of interest here
    - Authorization
  # Include the operation name in the access log record
  include_operation_name: true
  # Include the variables in the access log record
  include_variables: true
  # Include the payload in the access log record
  include_payload: true
  # Set to true to utilize async access-logging
  async: true
  # Set the buffer size of how many log entries can be buffered
  buffer_size: 1000
```

### Google Cloud Logging Configuration

To send access logs to Google Cloud Logging:

```yaml
access_logging:
  # Enable the feature
  enabled: true
  include_headers:
    # Include any headers of interest here
    - Authorization
  # Include the operation name in the access log record
  include_operation_name: true
  # Include the variables in the access log record
  include_variables: true
  # Include the payload in the access log record
  include_payload: false
  # Async is automatically disabled when using Google Cloud Logging
  # (GCP client handles batching internally, so this setting is ignored)
  async: false
  buffer_size: 1000
  # Google Cloud Logging configuration
  google_cloud_logging:
    # Enable Google Cloud Logging
    enabled: true
    # GCP Project ID (optional if GOOGLE_CLOUD_PROJECT env var is set)
    project_id: "my-gcp-project"
    # Log name in Google Cloud Logging (optional, defaults to "graphql-protect-access-logs")
    log_name: "graphql-protect-access-logs"
```

**Note**: When `google_cloud_logging.enabled: true`, the `async` option is automatically disabled (even if set to true). The GCP Cloud Logging client handles batching internally.

## How does it work?

For each operation we'll produce an access log record according to your provided configuration. 

If used in conjunction with persisted operations the access log will be produced after the operation is swapped for the payload, meaning you have full access to the operation name and payload.

If async is enabled, every access log record will be put on a channel and the logging is processed async. This way there is no waiting for slog to actually log the entry, but the request can be proxied immediately after. The amount of requests that can be buffered is configurable.
Metrics are available to see if you buffer overflows and logs are dropped, and how much of your configured buffer size is used. If this number is going up, you need to increase your buffersize.

## Environment Variables

When using Google Cloud Logging, the GCP Project ID can be provided via configuration or environment variables. The following priority order is used:

1. `google_cloud_logging.project_id` in configuration file
2. `GOOGLE_CLOUD_PROJECT` environment variable
3. `GCP_PROJECT` environment variable
4. `GCLOUD_PROJECT` environment variable

If no project ID is found through any of these methods, the server will fail to start with an error message.

## Metrics

Access logging provides the following Prometheus metrics:

### Async Mode (Stdout Only)

- `graphql_protect_access_logging_dropped_logs_total` - Counter of access log entries dropped due to full buffer
- `graphql_protect_access_logging_buffer_usage_current` - Current number of log entries in the async buffer
- `graphql_protect_access_logging_buffer_size_limit` - Maximum capacity of the async logging buffer

### Google Cloud Logging Mode

- `graphql_protect_access_logging_gcp_writes_total` - Total number of access log entries written to Google Cloud Logging
- `graphql_protect_access_logging_gcp_errors_total` - Total number of errors encountered while writing to Google Cloud Logging

These metrics help you monitor the health and performance of your access logging system.

## Important Notes

### Async Mode with Google Cloud Logging

Async mode is automatically disabled when Google Cloud Logging is enabled. If you set `async: true` with `google_cloud_logging.enabled: true`, the async setting will be ignored and a warning will be logged at startup:

```
Async mode is not supported with Google Cloud Logging - ignoring async setting (GCP client handles batching internally)
```

This ensures you only need to configure one flag (`google_cloud_logging.enabled`) to switch modes.

### Default Values

- **Log Name**: If not specified, defaults to `graphql-protect-access-logs`
- **Async**: Defaults to `false`
- **Buffer Size**: Defaults to `1000` (only relevant when async is enabled)
