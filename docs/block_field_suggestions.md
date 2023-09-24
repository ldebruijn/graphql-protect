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
  # The mask to apply whenever a field suggestion is found. The entire message will be replaced with this string
  mask: [redacted]
```

## How does it work?

We scan each `errors[].message` field in the responses and replace the message with a mask when we encounter a field suggestion.
