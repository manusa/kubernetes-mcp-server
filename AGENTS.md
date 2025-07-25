# AGENTS

This repository contains a Go implementation of a Model Context Protocol (MCP)
server for Kubernetes and OpenShift.

## Repository layout

- `cmd/kubernetes-mcp-server/` – entry point and Cobra CLI.
- `pkg/` – libraries grouped by domain (`kubernetes`, `mcp`, `output`, …).
- `Makefile` – tasks for building, formatting, linting and testing.
- `npm/` – Node packages that wrap the compiled binary. The `kubernetes-mcp-server` package
  defines a small launcher in `bin/index.js` which resolves the platform specific
  optional dependency (`kubernetes-mcp-server-<os>-<arch>`) and executes it.
- `python/` – Python package providing a script that downloads the correct
  platform binary from the GitHub releases page and runs it.

## Feature development

Implement new functionality in the Go sources under `cmd/` and `pkg/`.
The JavaScript and Python directories only wrap the compiled binary for
distribution (npm and PyPI). Most changes will not require touching them
unless the version or packaging needs to be updated.

## Building

Use the provided Makefile targets:

```bash
# Format source and build the binary
make build

# Build for all supported platforms
make build-all-platforms
```

`make build` will run `go fmt` and `go mod tidy` before compiling. The
resulting executable is `kubernetes-mcp-server`.

## Running

The README demonstrates running the server via
[`mcp-inspector`](https://modelcontextprotocol.io/docs/tools/inspector):

```bash
make build
npx @modelcontextprotocol/inspector@latest $(pwd)/kubernetes-mcp-server
```

The server is typically run locally and connects to the cluster using the same
configuration loading rules as `kubectl`. When started with `npx` or `uvx` it
downloads and executes a platform specific binary. The running process then
reads the kubeconfig resolved from the `--kubeconfig` flag, the `KUBECONFIG`
environment variable or the default `~/.kube/config` file. If those are not
present and the process executes inside a pod it falls back to in-cluster
configuration. This means that `npx kubernetes-mcp-server` on a workstation will
talk to whatever cluster your current kubeconfig points to (e.g. a local Kind
cluster).

## Tests

Run all Go tests with:

```bash
make test
```

The test suite relies on the `setup-envtest` tooling from
`sigs.k8s.io/controller-runtime`. The first run downloads a Kubernetes
`envtest` environment from the internet, so network access is required. Without
it some tests will fail during setup.

## Linting

Static analysis is performed with `golangci-lint`:

```bash
make lint
```

The `lint` target downloads the specified `golangci-lint` version if it is not
already present under `_output/tools/bin/`.

## Dependencies

When introducing new modules run `go mod tidy` (or `make tidy`) so that
`go.mod` and `go.sum` remain tidy.

## Coding style

- Go modules target Go **1.24** (see `go.mod`).
- Tests are written with the standard library `testing` package.
- Build, test and lint steps are defined in the Makefile—keep them working.

