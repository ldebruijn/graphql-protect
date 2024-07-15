# Exclude subgraph errors

Subgraph errors in a GraphQL server, though convenient, can pose risks. They can reveal internal details about the subgraph servers, potentially aiding malicious actors.


## Configuration

You can configure `graphql-protect` to exclude subgraph errors from your API.

```yaml
exclude_subgraph_errors:
  # Enable the feature, this will remove any field suggestions on your API
  enable: true #default
```

## How does it work?

If enabled the `errors[]` field in the responses is replaced with a "Subgraph errors redacted" message
