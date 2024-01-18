# Enforce POST

A rule that enforces the use of HTTP POST method when sending operations to the upstream GraphQL API.


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