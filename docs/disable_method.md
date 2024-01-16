# Disable Method

Add a rule that disables GET methods for being used to send operations to the GraphQL API.

<!-- TOC -->

## Configuration

You can configure `go-graphql-armor` to limit the maximum number of aliases allowed on an operation.

```yaml
disable_get_method:
  # Enable the feature
  enable: "true"
```

## Metrics

This rule produces metrics to help you gain insights into the behavior of the rule.

```
go_graphql_armor_disable_method_count{}
```

No metrics are produced when the rule is disabled or never encounters a GET request.