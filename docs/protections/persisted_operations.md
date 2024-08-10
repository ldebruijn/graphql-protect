# Persisted Operations

Persisted Operations are essentially an operation allowlist. Persisted Operations provide an additional layer of security to your GraphQL API by disallowing arbitrary queries to be performed against your APIs.

Check [Production Considerations](https://www.graphile.org/postgraphile/production/#simple-query-allowlist-persisted-queries--persisted-operations) for a more in-depth reasoning.

We recommend that all GraphQL APIs that only intend a specific/known set of clients to use the API should use Persisted Operations.

<!-- TOC -->

## Configuration

You can configure `graphql-protect` to enable Persisted Operations.

```yaml
# ...

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

# ...
```

## How it works

`graphql-protect` looks at the location specified for the `loader` and looks for any `*.json` files it can parse for persisted operations.
These loaders can be specified to look at local directories, or remote locations like GCP buckets.
`graphql-protect` will load the persisted operations from the location and update its internal state with any new operations.

## Loader

Currently we have support for the following loaders, specified by the `type` field in the loader configuration:

* `local` - load persisted operations from local file system, this is the default strategy. If need be this allows you to download files from an unsupported remote location to local storage, and have `graphql-protect` pick up on them.
* `gcp` - load persisted operations from a GCP bucket
* `noop` - no persisted operations are loaded. This is the strategy applied when an unknown type is supplied.

## Parsing Structure

To be able to parse Persisted Operations graphql-protect expects a `key-value` structure for `hash-operation` in the files.

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

## Why we don't support APQ

Automated Persisted Queries/Operations is essentially the same as Persisted Operations, except a client can send arbitrary operations which will be remembered by the server.

This completely removes the security benefit of Persisted Operations as any client can still send arbitrary operations. In fact, security is reduced since a malicious user could spam your endpoint with persisted operation registrations which would overflow your store and affect reliability.

For this reason we do not deem APQ a good practice, and have chosen not to support it.

## Generating Persisted Operations from the Client

In order to utilize this feature you need to generate the persisted operations that each client can perform.

[GraphQL Code Generator](https://the-guild.dev/graphql/codegen/plugins/presets/preset-client#persisted-documents)


## Metrics

This rule produces metrics to help you gain insights into the behavior of the rule.

```
graphql_protect_persisted_operations_result_count{state, result}
```

| `state`  | Description                                                                                                                                                   |
|---------|---------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `unknown` | The rule was not able to do its job. This happens either when `reject_on_failure` is set to `false` or the rule was not able to deserialize the request. |
| `error` | The rule caught an error during request body mutation.                                                                                                        |
| `known` | The rule received a hash for which it had a known operation                                                                                                   |


| `result`  | Description                   |
|---------|-------------------------------|
| `allowed` | The rule allowed the request  |
| `rejected` | The rule rejected the request |

```
graphql_protect_persisted_operations_load_result_count{type, result}
```


| `type`  | Description                   |
|---------|-------------------------------|
| `local` | Loaded using the local loader |
| `gcp`   | Loaded using the gcp loader   |
| `noop`  | Loaded using the noop loader  |


| `result`  | Description               |
|-----------|---------------------------|
| `success` | loading was successful    |
| `failure` | loading produced an error |

No metrics are produced when the rule is disabled.

```
graphql_protect_persisted_operations_unique_hashes_in_memory_count{}
```

No metrics are produced when the rule is disabled.
