FROM alpine:3.18

# Create new user
RUN addgroup -g 1001 -S go && \
    adduser -u 1001 -S go -G go && \
	mkdir /app

# Make sure we don't run as root
USER go

WORKDIR /app

COPY main /app/go-graphql-armor

EXPOSE 8080

ENTRYPOINT ["/app/go-graphql-armor"]
CMD ["serve"]