# Max depth

Max depth protections provide mechanisms for limiting the maximum field and nested lists.
Field depth restricts the depth of fields.
List depth restricts the amount of times lists can be nested.

Restricting the maximum depth of operations protect your API from abuse.

<!-- TOC -->

## Configuration

You can configure `graphql-protect` to limit the maximum depth on an operation.

```yaml
max_depth:
  # maximum field depth protections
  field:
    # Enable the protection
    enabled: false
    # The maximum depth allowed within a single document
    max: 1
    # Reject the document when the rule fails. Disable this to allow the document to be passed on to your API.
    reject_on_failure: false
  # maximum list depth protection, limits the depth of nested lists
  list:
    # Enable the protection
    enabled: false
    # The maximum depth allowed within a single document.
    max: 1
    # Reject the document when the rule fails. Disable this to allow the document to be passed on to your API.
    reject_on_failure: false
```

## Field protection

Ensures operations aren't too deep. Limiting this prevents excessive resolver calling, and waterfall processing tying up resources on your server.

The below field is an example operation that shows the depth of each field in the operation.
```graphql
{
    user { (1)
        address { (2)
            country { (3)
                contintent { (4)
                    planet { (5)
                        system { (6)
                            name  (7) 
                        }
                    }
                }
            }
        }
        pet {  (2)
            name (3)
        }       
    }
}
```

## List protection

Checks that lists aren't being nested too many times, leading to potential response amplification attacks
Ensures lists inside your operations aren't being nested too many times. Limiting this prevents potential response amplification attacks.

The below field is an example operation that shows the depth of each list in the operation.

```graphql
{ 
    user {
        friends { (1) 
            friends { (2) 
                friends { (3) 
                    friends { (4) 
                        friends { (5) 
                            name
                        } 
                    } 
                } 
            } 
        } 
    } 
}
```

Assuming each person has 100 friends, the above operation would yield `100 * 100 * 100 * 100 * 100` = `10.000.000.000` resources to be fetched.

## Metrics

This rule produces metrics to help you gain insights into the behavior of the rule.

```
graphql_protect_max_depth_results{type, result}
```

| `type`   | Description                                                                                                  |
|----------|--------------------------------------------------------------------------------------------------------------|
| `field`  | Field depth protection rule                                                                                  |
| `list`   | List depth protection rule                                                                                   |

| `result`  | Description                                                                                                  |
|---------|--------------------------------------------------------------------------------------------------------------|
| `allowed` | The rule condition succeeded                                                                                 |
| `rejected` | The rule condition failed and the request was rejected                                                       |
| `failed` | The rule condition failed but the request was not rejected. This happens when `reject_on_failure` is `false` |

No metrics are produced when the rule is disabled.