# Max depth

Restricting the maximum depth of operations that are allowed within a single operation to protect your API from abuse.

<!-- TOC -->

## Configuration

You can configure `go-graphql-armor` to limit the maximum depth allowed on an operation.

```yaml
max_depth:
  # Enable the feature
  enable: "true"
  # The maximum depth allowed within a single request.
  max: 15
  # Reject the request when the rule fails. Disable this to allow the request
  reject_on_failure: "true"
```

## Metrics

This rule produces metrics to help you gain insights into the behavior of the rule.

```
go_graphql_armor_max_depth_results{result}
```


| `result`  | Description                                                                                                  |
|---------|--------------------------------------------------------------------------------------------------------------|
| `allowed` | The rule condition succeeded                                                                                 |
| `rejected` | The rule condition failed and the request was rejected                                                       |
| `failed` | The rule condition failed but the request was not rejected. This happens when `reject_on_failure` is `false` |

No metrics are produced when the rule is disabled.