# Kubernetes

GraphQL Protect is intended to run as proxy to your main application. This allows it to scale with your application, and enjoys the benefit of loopback networking.

## Deployment resource

This specification describes a minimal example, focussing only on the elements relevant for GraphQL Protect. 

> [!NOTE]
> This is not a complete example, you're expected to mix this in with your existing deployment specification.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-graphql-api
spec:
  template:
    spec:
      containers:
      - name: my-graphql-api
        # Your main app
      - name: graphql-protect
        # Pin with specific version
        image: ghcr.io/ldebruijn/graphql-protect:latest
        args:
          # Override main command, specify mounted configuration file
          - "serve"
          - "-f"
          - "./config/graphql-protect-config.yml"
        ports:
          - containerPort: 8080
        readinessProbe: 
          periodSeconds: 1 
          initialDelaySeconds: 3 
          failureThreshold: 2 
          successThreshold: 2 
          httpGet:
            # Readiness probe for GraphQL Protect
            path: /internal/healthz/readiness
            port: 8080
          timeoutSeconds: 1
        env:
          - name: GOMAXPROCS
            valueFrom:
              resourceFieldRef:
                resource: requests.cpu
        volumeMounts:
          # Mount GraphQL Protect file in container local file system
          - mountPath: /app/config
            name: graphql-protect-config
            # Mount GraphQL Schema file in container local file system
          - mountPath: /app/schema
            name: schema-config
            # Mount empty dir in container local file system
          - mountPath: /app/store
            name: persisted-operations-store
      volumes:
        # Empty dir for storing persisted operations in
        - name: persisted-operations-store
          emptyDir: { }
        # GraphQL Protect configuration file yaml
        - name: graphql-protect-config
          configMap:
            name: graphql-protect-config
        # GraphQL schema file
        - name: schema.config
          configMap:
            name: graphql-schema-config
```

## Config Map Resource

You can create configmaps with the necessary configuration by running the following command and pointing it to your configuration file.

### Protect.yml

```shell
kubectl create configmap graphql-protect-config --from-file=protect.yml
```

### schema.graphql

```shell
kubectl create configmap graphql-schema-config --from-file=schema.graphql
```

> [!NOTE] 
> As always, make sure you're operating on the right context and namespace when executing these commands.