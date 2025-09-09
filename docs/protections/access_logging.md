# Access Logging

In some cases you want to keep a record of what operations were performed against your landscape. The access logging protection can provide that for you.
Access logging is done to STDOUT.

<!-- TOC -->

## Configuration

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

## How does it work?

For each operation we'll produce an access log record according to your provided configuration. 

If used in conjunction with persisted operations the access log will be produced after the operation is swapped for the payload, meaning you have full access to the operation name and payload.

If async is enabled, every access log record will be put on a channel and the logging is processed async. This way there is no waiting for slog to actually log the entry, but the request can be proxied immediately after. The amount of requests that can be buffered is configurable.
Metrics are available to see if you buffer overflows and logs are dropped, and how much of your configured buffer size is used. If this number is going up, you need to increase your buffersize.
