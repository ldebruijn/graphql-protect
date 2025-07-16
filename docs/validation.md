# Validation

GraphQL Protect offers the ability to run in CLI validation mode. In this mode it reads trusted documents and runs its ruleset against them. This is especially useful if you want to ensure trusted documents adhere to the ruleset limits you've defined before runtime.

## Configuration

The validation mode uses the same configuration structure as the HTTP proxy mode.
A specific example of this configuration is shown in the [Configuration chapter](configuration.md#graphql-protect---validate-run-mode)

## Output

The CLI outputs the results of running its ruleset in a structured format.
A rule can yield a `FAILED` or `REJECTED` result, depending on the configuration of that rule.
`FAILED` means the operation failed the rule, but due to the configuration the operation was still allowed.
`REJECTED` means the operation failed the rule and the operation was rejected as a result.

```text
+-------+-------------+----------------+--------------+----------------------+--------+
|     # | HASH        | OPERATIONNAME  | RULE         | ERROR                | RESULT |
+-------+-------------+----------------+--------------+----------------------+--------+
|     0 | i am a hash | operation name | example-rule | something went wrong | FAILED |
+-------+-------------+----------------+--------------+----------------------+--------+
| TOTAL | 1           |                |              |                      |        |
+-------+-------------+----------------+--------------+----------------------+--------+
```