# Max Batch

Restricts the maximum number of operations inside a batched request. This helps prevent an excessive number of operations reaching your landscape through minimal requests.
This can be useful to prevent DDoS attacks, Heap Overflows or Server overload.

<!-- TOC -->

## Configuration

You can configure `graphql-protect` to limit the maximum number of operations allowed inside a batch request.

```yaml
max_batch:
  # Enable the feature
  enable: true
  # The maximum number of operations within a single batched request.
  max: 5
  # Reject the request when the rule fails. Disable this to allow the request regardless of token count.
  reject_on_failure: true
```

## Metrics

This rule produces metrics to help you gain insights into the behavior of the rule.

```
graphql_protect_max_batch_results{result, size}
```
| `size`      | Description                                                                                                                               |
|-------------|-------------------------------------------------------------------------------------------------------------------------------------------|
| `{integer}` | The actual integer value of the observed size, only if the operations was allowed by the rule                                             |
| `exceeded`  | Whenever the operation was rejected or failed, `exceeded` is tracked. This is to prevent excessive metric generation upon malicious input |

| `result`            | Description                                                                                                  |
|---------------------|--------------------------------------------------------------------------------------------------------------|
| `allowed`           | The rule condition succeeded                                                                                 |
| `violated-rejected` | The rule condition failed and the request was rejected                                                       |
| `violated-allowed`  | The rule condition failed but the request was not rejected. This happens when `reject_on_failure` is `false` |

No metrics are produced when the rule is disabled.