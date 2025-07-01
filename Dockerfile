FROM golang:latest AS builder

WORKDIR /app

COPY ./ ./
RUN make build

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
WORKDIR /app
COPY --from=builder /app/kubernetes-mcp-server /app/kubernetes-mcp-server
ENTRYPOINT ["/app/kubernetes-mcp-server", "--port", "8080"]

EXPOSE 8080
