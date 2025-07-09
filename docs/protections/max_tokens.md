# Max Tokens

Restricting the maximum number of tokens in an operation helps prevent excessively large operations reaching your landscape.
This can be useful to prevent DDoS attacks, Heap Overflows or Server overload.

<!-- TOC -->

## Configuration

You can configure `graphql-protect` to limit the maximum number of tokens allowed on an operation.

```yaml
max_tokens:
  # Enable the feature
  enable: true
  # The maximum number of allowed tokens within a single request.
  max: 1000
  # Reject the request when the rule fails. Disable this to allow the request regardless of token count.
  reject_on_failure: true
  # Specific overrides for operations with the specified OperationName, and override max token value.
  overrides:
    MyOperation: 2000
    AnotherOperation: 5000
```

### Overrides

It's possible to override the max token limit for specific operations.
This is useful when you have a few operations that are known to be larger than the default limit.

```yaml
max_tokens:
  enable: true
  max: 1000
  reject_on_failure: true
  
```

In this example, the operation `MyOperation` will have a max token limit of 2000, and `AnotherOperation` will have a limit of 5000.
All other operations will have the default limit of 1000.

The limit specified by the override can be both larger and smaller than the general `max` limit.

## Metrics

This rule produces metrics to help you gain insights into the behavior of the rule.

```
graphql_protect_max_tokens_results{result}
```


| `result`  | Description                                                                                                  |
|---------|--------------------------------------------------------------------------------------------------------------|
| `allowed` | The rule condition succeeded                                                                                 |
| `rejected` | The rule condition failed and the request was rejected                                                       |
| `failed` | The rule condition failed but the request was not rejected. This happens when `reject_on_failure` is `false` |

No metrics are produced when the rule is disabled.