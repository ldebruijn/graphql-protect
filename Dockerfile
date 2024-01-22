FROM alpine:3.19

ARG BUILD_DATE
ARG VERSION
ARG REVISION

LABEL org.opencontainers.image.title=graphql-protect \
	org.opencontainers.image.description="A dead-simple yet highly customizable security sidecar compatible with any HTTP GraphQL Server or Gateway." \
	org.opencontainers.image.created=$BUILD_DATE \
	org.opencontainers.image.authors=ldebruijn \
	org.opencontainers.image.url=https://github.com/ldebruijn/graphql-protect \
	org.opencontainers.image.documentation=https://github.com/ldebruijn/graphql-protect \
	org.opencontainers.image.source=https://github.com/ldebruijn/graphql-protect \
	org.opencontainers.image.version=$VERSION \
	org.opencontainers.image.revision=$REVISION \
	org.opencontainers.image.licenses=MIT \
	org.opencontainers.image.base.name=alpine

# Create new user
RUN addgroup -g 1001 -S go && \
    adduser -u 1001 -S go -G go && \
	mkdir /app

# Make sure we don't run as root
USER go

WORKDIR /app

COPY main /app/graphql-protect

EXPOSE 8080

ENTRYPOINT ["/app/graphql-protect"]
CMD ["serve"]