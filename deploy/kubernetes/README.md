# Kubernetes MCP Server Deployment

This guide explains how to deploy the "Kubernetes MCP Server" to a Kubernetes cluster.

## Prerequisites

1.  **`envsubst`:** A utility for substituting environment variables in shell-format strings.
    *   Verify with: `envsubst --version`


## Deployment Steps

1.  **Navigate to the Project Directory:**
    Open your terminal and change to the root directory of this project where the `Makefile` is located.

1.  **Set the Container Image (Optional):**
    The `Makefile` is designed to use a `CONTAINER_IMAGE` variable.
    *   **Default Image:** If not specified, it defaults to `quay.io/foo/kubernetes-mcp-server:latest`.
    *   **Override Image:** You can specify a different container image and tag by setting the `CONTAINER_IMAGE` variable when running `make`.   

1. **Build the Image:**
This command builds the container image and pushes it into the registry.

    ```bash
    make image
    ```

1.  **Deploy to Kubernetes:**
    Run the following command:
    ```bash
    make kube-deploy
    ```
    Or, if overriding the image:
    ```bash
    make kube-deploy CONTAINER_IMAGE=your-registry/your-image-name:your-tag
    ```

## Verifying the Deployment

Once the deployment is complete, you can verify that the application is running:

1.  **Check Pods:**
    The application runs in the `mcp-system` namespace.
    ```bash
    kubectl get pods -n mcp-system
    ```
    You should see a pod with a name like `kubernetes-mcp-server-xxxxxxxxxx-xxxxx` in a `Running` state.

2.  **Check Service:**
    ```bash
    kubectl get svc -n mcp-system
    ```
    You should see the `kubernetes-mcp-server` service listed.

3.  **View Logs:**
    ```bash
    kubectl logs -n mcp-system -l app=kubernetes-mcp-server -f
    ```

## Accessing the Application

The `kubernetes-mcp-server` service is typically exposed within the cluster. To access it from your local machine, you can use `kubectl port-forward`.

**Port Forwarding:**

1.  Open a new terminal window.
2.  Run the following command to forward a local port (e.g., 8081) to the service's port (8080):
    ```bash
    kubectl port-forward svc/kubernetes-mcp-server -n mcp-system 8081:8080
    ```
    *   `8081:8080`: Maps local port `8081` to the service's target port `8080`.

    Keep this terminal window open. While `kubectl port-forward` is running, your server will be accessible on `http://localhost:8081`.


## Goose SSE Configuration

If you are using [Goose](https://block.github.io/goose/) to connect to Server-Sent Events (SSE) provided by the `kubernetes-mcp-server`, you can configure it as follows.

```yaml
extensions:
  kubernetes-remote:
    description: null
    enabled: true
    envs: {}
    name: kubernetes-remote
    timeout: 200 # Timeout in seconds for the SSE connection
    type: sse
    uri: http://localhost:8081/sse # Points to the local port forwarded to the k8s service
```