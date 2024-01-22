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
  enabled: "true"
  # Fail unknown operations, disable this feature to allow unknown operations to reach your GraphQL API
  reject_on_failure: "true"
  # Store is the location on local disk where go-graphql-armor can find the persisted operations, it loads any `*.json` files on disk
  store: "./store"
  reload:
    enabled: "true"
    # The interval in which the local store dir is read and refreshes the internal state
    interval: 5m
    # The timeout for the remote operation
    timeout: 10s
  remote:
    # Load persisted operations from a GCP Cloud Storage bucket.
    # Will look at all the objects in the bucket and try to load any object with a `.json` extension
    gcp_bucket: "gs://somebucket"

# ...
```

## How it works

`go-graphql-armor` looks at the `store` location on local disk to find any `*.json` files it can parse for persisted operations. 

It can be configured to look at this directory and reload based on the files on local disk.

Additionally, it can be configured to fetch operations from a remote location onto the local disk.

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

## Why we don't support APQ

Automated Persisted Queries/Operations is essentially the same as Persisted Operations, except a client can send arbitrary operations which will be remembered by the server.

This completely removes the security benefit of Persisted Operations as any client can stills end arbitrary operations. In fact, security is reduced since a malicious user could spam your endpoint with persisted operation registrations which would overflow your store and affect reliability.

For this reason we do not deem APQ a good practice, and have chosen not to support it.

## Generating Persisted Operations from the Client

In order to utilize this feature you need to generate the persisted operations that each client can perform.

[GraphQL Code Generator](https://the-guild.dev/graphql/codegen/plugins/presets/preset-client#persisted-documents)


## Metrics

This rule produces metrics to help you gain insights into the behavior of the rule.

```
go_graphql_armor_persisted_operations_results{state, result}
```

| `state`  | Description                                                                                                                                                   |
|---------|---------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `unknown` | The rule was not able to do its job. This happens either when `fail_unknown_operations` is set to `false` or the rule was not able to deserialize the request. |
| `error` | The rule caught an error during request body mutation.                                                                                                        |
| `known` | The rule received a hash for which it had a known operation                                                                                                   |


| `result`  | Description                   |
|---------|-------------------------------|
| `allowed` | The rule allowed the request  |
| `rejected` | The rule rejected the request |

```
go_graphql_armor_persisted_operations_reload{system}
```


| `system` | Description                                                                                           |
|--------|-------------------------------------------------------------------------------------------------------|
| `local`  | The rule reloaded its state from local storage                                                        |
| `remote` | The rule reloaded the remote state onto local disk. This does not refresh the local state on its own. |

No metrics are produced when the rule is disabled.
