# Schema

`go-graphql-armor` needs to know your schema in order to perform its validations. 

<!-- TOC -->

## Configuration

```yaml
# ...

schema:
  # Path to a local file in which the schema can be found
  path: "./schema.graphql"
  # Automatically reload the schema file. 
  # It will reload the contents of the file referenced by the `schema.path` configuration option
  # after each `schema.auto_reload.interval` has passed.
  auto_reload:
    # Enable automatic file reloading
    enabled: "true"
    # The interval in which the schema file should be reloaded
    interval: 5m
```

## Metrics

```
go_graphql_armor_schema_reload{state}
```

| `state`   | Description                                                 |
|-----------|-------------------------------------------------------------|
| `failed`  | Reloading the file from local disk has failed               |
| `success` | The schema file was successfully reloaded from local disk   |
