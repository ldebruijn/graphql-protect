# Max Aliases

Restricting the maximum number of aliases that are allowed within a single operation protects your API from Brute Force attacks.

Aliases allow you to perform the same operation multiple times, within a single request. This opens up the possibility of for example trying out login operations 1000 times with 1 request. 
Or even worse, uploading a 1 MB image with 1000 aliases in 1 request using the same binary data, essentially creating a Denial of Service attack on your API with 1MB of data, resulting in 1GB of data processed on the server.

<!-- TOC -->

## Configuration

You can configure `go-graphql-armor` to limit the maximum number of aliases allowed on an operation.

```yaml
max_aliases:
  # Enable the feature
  enable: true
  # The maximum number of allowed aliases within a single request.
  max: 15
  # Reject the request when the rule fails. Disable this to allow the request
  reject_on_failure: true
```

## Metrics

This rule produces metrics to help you gain insights into the behavior of the rule.

```
go_graphql_armor_max_aliases_results{result}
```


| `result`  | Description                                                                                                  |
|---------|--------------------------------------------------------------------------------------------------------------|
| `allowed` | The rule condition succeeded                                                                                 |
| `rejected` | The rule condition failed and the request was rejected                                                       |
| `failed` | The rule condition failed but the request was not rejected. This happens when `reject_on_failure` is `false` |

No metrics are produced when the rule is disabled.