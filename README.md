# Kubernetes MCP Server

[![GitHub License](https://img.shields.io/github/license/manusa/kubernetes-mcp-server)](https://github.com/manusa/kubernetes-mcp-server/blob/main/LICENSE)
[![npm](https://img.shields.io/npm/v/kubernetes-mcp-server)](https://www.npmjs.com/package/kubernetes-mcp-server)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/manusa/kubernetes-mcp-server?sort=semver)](https://github.com/manusa/kubernetes-mcp-server/releases/latest)
[![Build](https://github.com/manusa/kubernetes-mcp-server/actions/workflows/build.yaml/badge.svg)](https://github.com/manusa/kubernetes-mcp-server/actions/workflows/build.yaml)

[✨ Features](#features) | [🚀 Getting Started](#getting-started) | [🎥 Demos](#demos) | [⚙️ Configuration](#configuration) | [🧑‍💻 Development](#development)

https://github.com/user-attachments/assets/be2b67b3-fc1c-4d11-ae46-93deba8ed98e

## ✨ Features <a id="features"></a>

A powerful and flexible Kubernetes [Model Context Protocol (MCP)](https://blog.marcnuri.com/model-context-protocol-mcp-introduction) server implementation with support for **Kubernetes** and **OpenShift**.

- **✅ Configuration**:
  - Automatically detect changes in the Kubernetes configuration and update the MCP server.
  - **View** and manage the current [Kubernetes `.kube/config`](https://blog.marcnuri.com/where-is-my-default-kubeconfig-file) or in-cluster configuration.
- **✅ Generic Kubernetes Resources**: Perform operations on **any** Kubernetes or OpenShift resource.
  - Any CRUD operation (Create or Update, Get, List, Delete).
- **✅ Pods**: Perform Pod-specific operations.
  - **List** pods in all namespaces or in a specific namespace.
  - **Get** a pod by name from the specified namespace.
  - **Delete** a pod by name from the specified namespace.
  - **Show logs** for a pod by name from the specified namespace.
  - **Exec** into a pod and run a command.
  - **Run** a container image in a pod and optionally expose it.
- **✅ Namespaces**: List Kubernetes Namespaces.
- **✅ Events**: View Kubernetes events in all namespaces or in a specific namespace.
- **✅ Projects**: List OpenShift Projects.

Unlike other Kubernetes MCP server implementations, this **IS NOT** just a wrapper around `kubectl` or `helm` command-line tools.

There is **NO NEED** for external dependencies or tools to be installed on the system.
If you're using the native binaries you don't even need to have Node or Python installed on your system.

## 🚀 Getting Started <a id="getting-started"></a>

### Requirements

- Access to a Kubernetes cluster.

### Claude Desktop

#### Using npx

If you have npm installed, this is the fastest way to get started with `kubernetes-mcp-server` on Claude Desktop.

Open your `claude_desktop_config.json` and add the mcp server to the list of `mcpServers`:
``` json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": [
        "-y",
        "kubernetes-mcp-server@latest"
      ]
    }
  }
}
```

### VS Code / VS Code Insiders

Install the Kubernetes MCP server extension in VS Code Insiders by pressing the following link:

[<img alt="Install in VS Code Insiders" src="https://img.shields.io/badge/VS_Code_Insiders-VS_Code_Insiders?style=flat-square&label=Install%20Server&color=24bfa5">](https://insiders.vscode.dev/redirect?url=vscode-insiders%3Amcp%2Finstall%3F%257B%2522name%2522%253A%2522kubernetes%2522%252C%2522command%2522%253A%2522npx%2522%252C%2522args%2522%253A%255B%2522-y%2522%252C%2522kubernetes-mcp-server%2540latest%2522%255D%257D)

Alternatively, you can install the extension manually by running the following command:

```shell
# For VS Code
code --add-mcp '{"name":"kubernetes","command":"npx","args":["kubernetes-mcp-server@latest"]}'
# For VS Code Insiders
code-insiders --add-mcp '{"name":"kubernetes","command":"npx","args":["kubernetes-mcp-server@latest"]}'
```

### Goose CLI

[Goose CLI](https://blog.marcnuri.com/goose-on-machine-ai-agent-cli-introduction) is the easiest (and cheapest) way to get rolling with artificial intelligence (AI) agents.

#### Using npm

If you have npm installed, this is the fastest way to get started with `kubernetes-mcp-server`.

Open your goose `config.yaml` and add the mcp server to the list of `mcpServers`:
```yaml
extensions:
  kubernetes:
    command: npx
    args:
      - -y
      - kubernetes-mcp-server@latest

```

## Tools

### `configuration_view`

Get the current Kubernetes configuration content as a kubeconfig YAML

**Parameters:**
- `minified` (`boolean`, optional, default: `true`)
  - Return a minified version of the configuration
  - If `true`, keeps only the current-context and relevant configuration pieces
  - If `false`, returns all contexts, clusters, auth-infos, and users

### `events_list`

List all the Kubernetes events in the current cluster from all namespaces

**Parameters:**
- `namespace` (`string`, optional)
  - Namespace to retrieve the events from. If not provided, will list events from all namespaces

### `namespaces_list`

List all the Kubernetes namespaces in the current cluster

**Parameters:** None

### `pods_delete`

Delete a Kubernetes Pod in the current or provided namespace with the provided name

**Parameters:**
- `name` (`string`, required)
  - Name of the Pod to delete
- `namespace` (`string`, required)
  - Namespace to delete the Pod from

### `pods_exec`

Execute a command in a Kubernetes Pod in the current or provided namespace with the provided name and command

**Parameters:**
- `command` (`string[]`, required)
  - Command to execute in the Pod container
  - First item is the command, rest are arguments
  - Example: `["ls", "-l", "/tmp"]`
- `name` (string, required)
  - Name of the Pod
- `namespace` (string, required)
  - Namespace of the Pod

### `pods_get`

Get a Kubernetes Pod in the current or provided namespace with the provided name

**Parameters:**
- `name` (`string`, required)
  - Name of the Pod
- `namespace` (`string`, required)
  - Namespace to get the Pod from

### `pods_list`

List all the Kubernetes pods in the current cluster from all namespaces

**Parameters:** None

### `pods_list_in_namespace`

List all the Kubernetes pods in the specified namespace in the current cluster

**Parameters:**
- `namespace` (`string`, required)
  - Namespace to list pods from

### `pods_log`

Get the logs of a Kubernetes Pod in the current or provided namespace with the provided name

**Parameters:**
- `name` (`string`, required)
  - Name of the Pod to get logs from
- `namespace` (`string`, required)
  - Namespace to get the Pod logs from
- `container` (`string`, optional)
  - Name of the Pod container to get logs from

### `pods_run`

Run a Kubernetes Pod in the current or provided namespace with the provided container image and optional name

**Parameters:**
- `image` (`string`, required)
  - Container Image to run in the Pod
- `namespace` (`string`, required)
  - Namespace to run the Pod in
- `name` (`string`, optional)
  - Name of the Pod (random name if not provided)
- `port` (`number`, optional)
  - TCP/IP port to expose from the Pod container
  - No port exposed if not provided

### `resources_create_or_update`

Create or update a Kubernetes resource in the current cluster by providing a YAML or JSON representation of the resource

**Parameters:**
- `resource` (`string`, required)
  - A JSON or YAML containing a representation of the Kubernetes resource
  - Should include top-level fields such as apiVersion, kind, metadata, and spec

**Common apiVersion and kind include:**
- v1 Pod
- v1 Service
- v1 Node
- apps/v1 Deployment
- networking.k8s.io/v1 Ingress

### `resources_delete`

Delete a Kubernetes resource in the current cluster

**Parameters:**
- `apiVersion` (`string`, required)
  - apiVersion of the resource (e.g., `v1`, `apps/v1`, `networking.k8s.io/v1`)
- `kind` (`string`, required)
  - kind of the resource (e.g., `Pod`, `Service`, `Deployment`, `Ingress`)
- `name` (`string`, required)
  - Name of the resource
- `namespace` (`string`, optional)
  - Namespace to delete the namespaced resource from
  - Ignored for cluster-scoped resources
  - Uses configured namespace if not provided

### `resources_get`

Get a Kubernetes resource in the current cluster

**Parameters:**
- `apiVersion` (`string`, required)
  - apiVersion of the resource (e.g., `v1`, `apps/v1`, `networking.k8s.io/v1`)
- `kind` (`string`, required)
  - kind of the resource (e.g., `Pod`, `Service`, `Deployment`, `Ingress`)
- `name` (`string`, required)
  - Name of the resource
- `namespace` (`string`, optional)
  - Namespace to retrieve the namespaced resource from
  - Ignored for cluster-scoped resources
  - Uses configured namespace if not provided

### `resources_list`

List Kubernetes resources and objects in the current cluster

**Parameters:**
- `apiVersion` (`string`, required)
  - apiVersion of the resources (e.g., `v1`, `apps/v1`, `networking.k8s.io/v1`)
- `kind` (`string`, required)
  - kind of the resources (e.g., `Pod`, `Service`, `Deployment`, `Ingress`)
- `namespace` (`string`, optional)
  - Namespace to retrieve the namespaced resources from
  - Ignored for cluster-scoped resources
  - Lists resources from all namespaces if not provided

## 🎥 Demos <a id="demos"></a>

### Diagnosing and automatically fixing an OpenShift Deployment

Demo showcasing how Kubernetes MCP server is leveraged by Claude Desktop to automatically diagnose and fix a deployment in OpenShift without any user assistance.

https://github.com/user-attachments/assets/a576176d-a142-4c19-b9aa-a83dc4b8d941

### _Vibe Coding_ a simple game and deploying it to OpenShift

In this demo, I walk you through the process of _Vibe Coding_ a simple game using VS Code and how to leverage [Podman MCP server](https://github.com/manusa/podman-mcp-server) and Kubernetes MCP server to deploy it to OpenShift.

<a href="https://www.youtube.com/watch?v=l05jQDSrzVI" target="_blank">
 <img src="docs/images/vibe-coding.jpg" alt="Vibe Coding: Build & Deploy a Game on Kubernetes" width="240"  />
</a>

## ⚙️ Configuration <a id="configuration"></a>

The Kubernetes MCP server can be configured using command line (CLI) arguments.

You can run the CLI executable either by using `npx` or by downloading the [latest release binary](https://github.com/manusa/kubernetes-mcp-server/releases/latest).

```shell
# Run the Kubernetes MCP server using npx (in case you have npm installed)
npx kubernetes-mcp-server@latest --help
```

```shell
# Run the Kubernetes MCP server using the latest release binary
./kubernetes-mcp-server --help
```

### Configuration Options

| Option        | Description                                                                                                                                                                                                                                                                                   |
|---------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `--sse-port`  | Starts the MCP server in Server-Sent Event (SSE) mode and listens on the specified port.                                                                                                                                                                                                      |
| `--log-level` | Sets the logging level (values [from 0-9](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md)). Similar to [kubectl logging levels](https://kubernetes.io/docs/reference/kubectl/quick-reference/#kubectl-output-verbosity-and-debugging). |

## 🧑‍💻 Development <a id="development"></a>

### Running with mcp-inspector

Compile the project and run the Kubernetes MCP server with [mcp-inspector](https://modelcontextprotocol.io/docs/tools/inspector) to inspect the MCP server.

```shell
# Compile the project
make build
# Run the Kubernetes MCP server with mcp-inspector
npx @modelcontextprotocol/inspector@latest $(pwd)/kubernetes-mcp-server
```
