# HTTP Configuration

## HTTP server configuration

```yaml
web:
  # Maximum duration to read the entire request
  read_timeout: 5s
  # Maximum duration before timing out writes of the response
  write_timeout: 10s
  # Maximum time to wait between idle requests for keep alive
  idle_timeout: 120s
  # Time to wait until forcibly shutting down protect, after receiving a shutdown signal
  shutdown_timeout: 20s
  # host and port to listen on
  host: 0.0.0.0:8080
  # path that receives GraphQL traffic
  path: /graphql
  # limit the maximum size of a request body that is allowed
  # this helps prevent OOM attacks through excessively large request payloads.
  # A limit of `0` disables this protection.
  request_body_max_bytes: 102400

target:
  # Target host and port to send traffic to after validating
  host: http://localhost:8081
  # Dial timeout waiting for a connection to complete with the target upstream
  timeout: 10s
  # Interval of keep alive probes
  keep_alive: 180s
  tracing:
    # Headers to redact when sending tracing information
    redacted_headers: []
```

## HTTP Request Body Max Byte size

To prevent OOM attacks through excessively large request bodies, a default limit is posed on request body size of `100kb`. This limit is generally speaking ample space for GraphQL request bodies, while also providing solid protections.

You can modify this limit by changing the following configuration option

```yaml
web:
  # limit the maximum size of a request body that is allowed
  # this helps prevent OOM attacks through excessively large request payloads.
  # A limit of `0` disables this protection.
  request_body_max_bytes: 102400
```

### Metrics

A metric is exposed to track if and when a request is rejected that exceeds this limit.

```
graphql_protect_http_request_max_body_bytes_exceeded_count{}
```

No metrics are produced for requests that do not exceed this limit.