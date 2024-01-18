# Enforce POST

A rule that enforces the use of HTTP POST method when sending operations to the upstream GraphQL API.

The rule will block requests with non-POST HTTP methods **only** if the requests contain GraphQL operations. If no operation is found it will still forward the request to the upstream. This is useful for accessing GraphiQL for example through Go GraphQL Armor.


<!-- TOC -->

## Configuration

```yaml
enforce_post:
  # Enable the feature
  enable: "true"
```

## Metrics

This rule produces metrics to help you gain insights into the behavior of the rule.

```
go_graphql_armor_enforce_post_count{}
```

No metrics are produced when the rule is disabled or never encounters operations through a non-POST request.