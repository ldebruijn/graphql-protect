# Documentation

Please see each section for in-depth documentation

## Configuration

[This section](configuration.md) describes the configuration options for GraphQL Protect.

## Run modes

Protect supports various running modes for different needs and purposes.

* `serve` runs as an HTTP proxy protection your GraphQL during runtime. Check out the [Deployment Options](#run) section for more configuration options
* `validate` runs as a CLI tool, validating your Persisted Operations against your schema and configured protections (see [this page](configuration.md#graphql-protect---validate-run-mode) for more info how to set this up)
* `version` outputs versioning info of protect

If no runmode is explicitly specified, `serve` is assumed as default

## Protections

This section contains all the documentation about each protection feature.

* [Persisted Operations](protections/persisted_operations.md)
* [Block Field Suggestions](protections/block_field_suggestions.md)
* [Max Aliases](protections/max_aliases.md)
* [Max Tokens](protections/max_tokens.md)
* [Enforce POST](protections/enforce_post.md)
* [Max Batch](protections/max_batch.md)
* [Access Logging](protections/access_logging.md)


## Run

This section contains in depth documentation for run strategies

* [Kubernetes](run/kubernetes.md)
* [Docker](run/docker.md)
* [Tracing / OpenTelemetry](run/tracing.md)