# Kubernetes MCP Server

[![GitHub License](https://img.shields.io/github/license/manusa/kubernetes-mcp-server)](https://github.com/manusa/kubernetes-mcp-server/blob/main/LICENSE)
[![npm](https://img.shields.io/npm/v/kubernetes-mcp-server)](https://www.npmjs.com/package/kubernetes-mcp-server)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/manusa/kubernetes-mcp-server?sort=semver)](https://github.com/manusa/kubernetes-mcp-server/releases/latest)
[![Build](https://github.com/manusa/kubernetes-mcp-server/actions/workflows/build.yaml/badge.svg)](https://github.com/manusa/kubernetes-mcp-server/actions/workflows/build.yaml)

[✨ Features](#features) | [🚀 Getting Started](#getting-started) | [🎥 Demos](#demos)

https://github.com/user-attachments/assets/be2b67b3-fc1c-4d11-ae46-93deba8ed98e

## ✨ Features <a id="features"></a>

A powerful and flexible Kubernetes MCP server implementation with support for OpenShift.

- **✅ Configuration**: View and manage the [Kubernetes `.kube/config`](https://blog.marcnuri.com/where-is-my-default-kubeconfig-file).
  - **View** the current configuration.
- **✅ Generic Kubernetes Resources**: Perform operations on any Kubernetes resource.
  - Any CRUD operation (Create or Update, Get, List, Delete).
- **✅ Pods**: Perform Pod-specific operations.
  - **List** pods in all namespaces or in a specific namespace.
  - **Get** a pod by name from the specified namespace.
  - **Delete** a pod by name from the specified namespace.
  - **Show logs** for a pod by name from the specified namespace.
  - **Run** a container image in a pod and optionally expose it.

## 🚀 Getting Started <a id="getting-started"></a>

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

## 🎥 Demos <a id="demos"></a>

### Diagnosing and automatically fixing an OpenShift Deployment

Demo showcasing how Kubernetes MCP server is leveraged by Claude Desktop to automatically diagnose and fix a deployment in OpenShift without any user assistance.

https://github.com/user-attachments/assets/a576176d-a142-4c19-b9aa-a83dc4b8d941

