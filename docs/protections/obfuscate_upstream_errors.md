# Obfuscate upstream errors

Upstream errors in a GraphQL server, though convenient, can pose risks. They can reveal internal details about the upstream server(s), potentially aiding malicious actors.


## Configuration

You can configure `graphql-protect` to exclude upstream errors from your API.

```yaml
# Configures if upstream errors need to be obfuscated, this can help you hide internals of your upstream landscape

obfuscate_upstream_errors: true # default
```

## How does it work?

If enabled the `errors[].message` field in the response is replaced with an `"Error(s) redacted" message`
