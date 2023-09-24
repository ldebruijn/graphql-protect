# Persisted Operations

Persisted Operations are essentially an operation allowlist. Persisted Operations provide an additional layer of security to your GraphQL API by disallowing arbitrary queries to be performed against your APIs.

Check [Production Considerations](https://www.graphile.org/postgraphile/production/#simple-query-allowlist-persisted-queries--persisted-operations) for a more in-depth reasoning.

We recommend that all GraphQL APIs that only intend a specific/known set of clients to use the API should use Persisted Operations.

<!-- TOC -->

## Configuration

You can configure `go-graphql-armor` to enable Persisted Operations.

```yaml
# ...

persisted_operations:
  # Enable or disable the feature, enabled by default
  enabled: true
  # Fail unknown operations, disable this feature to allow unknown operations to reach your GraphQL API
  fail_unknown_operations: true
  # Determines the strategy for loading the supported operations.
  # Only one store will be used
  store:
    # Load persisted operations from a directory on the local filesystem. 
    # Will look at all files in the directory and attempt to load any file with a `.json` extension
    dir: "./my-dir"
    # Load persisted operations from a GCP Cloud Storage bucket.
    # Will look at all the objects in the bucket and try to load any object with a `.json` extension
    gcp_bucket: "gs://somebucket"

# ...
```

## Parsing Structure

To be able to parse Persisted Operations go-graphql-armor expects a `key-value` structure for `hash-operation` in the files.

`any-file.json`
```json
{
  "key": "query { product(id: 1) { id name } }",
  "another-key": "query { hello }"
}
```

Once loaded, any incoming operation with a known hash will be modified to include the operations specified as the value.

## Request Structure

We follow the [APQ specification](https://github.com/apollographql/apollo-link-persisted-queries#apollo-engine) for **sending** hashes to the server.

> **Important:**
> While we use the specification for APQ, be aware that _automatically_ persisting unknown operations is **NOT** supported.
