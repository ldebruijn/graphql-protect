# Docker

GraphQL Protect is intended to run as sidecar to your main application. This allows it to scale with your application, and enjoys the benefit of loopback networking.

## Setting up

#### Pull

```shell
docker pull ghcr.io/ldebruijn/graphql-protect:latest
```

#### Run

```shell
docker run -p 8080:8080 -v $(pwd)/protect.yml:/app/protect.yml -v $(pwd)/schema.graphql:/app/schema.graphql ghcr.io/ldebruijn/graphql-protect:latest
```

This mounts the necessary configuration and schema files from your local filesystem onto your container, and exposes port 8080 to the host machine.

## Networking

If you want to [reach a process on the host machine](https://docs.docker.com/desktop/networking/#i-want-to-connect-from-a-container-to-a-service-on-the-host), be sure to use `host.docker.internal` as the hostname for the proxy target, instead of `localhost`.

If you want to reach another container, be sure the two containers are in the [same container network](https://docs.docker.com/network/) to be able to communicate with each other.