# Field Suggestions

Field suggestions in a GraphQL server, though convenient, can pose risks. They can reveal internal details, like field or operation names, potentially aiding malicious actors.

Disabling field suggestions prevent the discovery of your GraphQL schema even when Introspection is disabled.

<!-- TOC -->

## Configuration

You can configure `go-graphql-armor` to remove field suggestions from your API.

```yaml
block_field_suggestions:
  # Enable the feature, this will remove any field suggestions on your API
  enable: true
```

## How does it work?

We scan each `errors[].message` field and replace the message with a mask when we encounter a field suggestion.
