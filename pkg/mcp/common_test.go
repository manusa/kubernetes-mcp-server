package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1spec "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	toolswatch "k8s.io/client-go/tools/watch"
	"k8s.io/utils/ptr"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/env"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/remote"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/store"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/versions"
	"sigs.k8s.io/controller-runtime/tools/setup-envtest/workflows"
	"testing"
	"time"
)

// envTest has an expensive setup, so we only want to do it once per entire test run.
var envTest *envtest.Environment
var envTestRestConfig *rest.Config
var envTestUser = envtest.User{Name: "test-user", Groups: []string{"test:users"}}

func TestMain(m *testing.M) {
	// Set up
	envTestDir, err := store.DefaultStoreDir()
	if err != nil {
		panic(err)
	}
	envTestEnv := &env.Env{
		FS:  afero.Afero{Fs: afero.NewOsFs()},
		Out: os.Stdout,
		Client: &remote.HTTPClient{
			IndexURL: remote.DefaultIndexURL,
		},
		Platform: versions.PlatformItem{
			Platform: versions.Platform{
				OS:   runtime.GOOS,
				Arch: runtime.GOARCH,
			},
		},
		Version: versions.AnyVersion,
		Store:   store.NewAt(envTestDir),
	}
	envTestEnv.CheckCoherence()
	workflows.Use{}.Do(envTestEnv)
	versionDir := envTestEnv.Platform.Platform.BaseName(*envTestEnv.Version.AsConcrete())
	envTest = &envtest.Environment{
		BinaryAssetsDirectory: filepath.Join(envTestDir, "k8s", versionDir),
	}
	adminSystemMasterBaseConfig, _ := envTest.Start()
	au, err := envTest.AddUser(envTestUser, adminSystemMasterBaseConfig)
	if err != nil {
		panic(err)
	}
	envTestRestConfig = au.Config()

	//Create test data as administrator
	ctx := context.Background()
	restoreAuth(ctx)
	createTestData(ctx)

	// Test!
	code := m.Run()

	// Tear down
	if envTest != nil {
		_ = envTest.Stop()
	}
	os.Exit(code)
}

type mcpContext struct {
	ctx           context.Context
	tempDir       string
	cancel        context.CancelFunc
	mcpServer     *Server
	mcpHttpServer *httptest.Server
	mcpClient     *client.Client
}

func (c *mcpContext) beforeEach(t *testing.T) {
	var err error
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.tempDir = t.TempDir()
	c.withKubeConfig(nil)
	if c.mcpServer, err = NewSever(Configuration{}); err != nil {
		t.Fatal(err)
		return
	}
	c.mcpHttpServer = server.NewTestServer(c.mcpServer.server)
	if c.mcpClient, err = client.NewSSEMCPClient(c.mcpHttpServer.URL + "/sse"); err != nil {
		t.Fatal(err)
		return
	}
	if err = c.mcpClient.Start(c.ctx); err != nil {
		t.Fatal(err)
		return
	}
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{Name: "test", Version: "1.33.7"}
	_, err = c.mcpClient.Initialize(c.ctx, initRequest)
	if err != nil {
		t.Fatal(err)
		return
	}
}

func (c *mcpContext) afterEach() {
	c.cancel()
	c.mcpServer.Close()
	_ = c.mcpClient.Close()
	c.mcpHttpServer.Close()
}

func testCase(t *testing.T, test func(c *mcpContext)) {
	mcpCtx := &mcpContext{}
	mcpCtx.beforeEach(t)
	defer mcpCtx.afterEach()
	test(mcpCtx)
}

// withKubeConfig sets up a fake kubeconfig in the temp directory based on the provided rest.Config
func (c *mcpContext) withKubeConfig(rc *rest.Config) *api.Config {
	fakeConfig := api.NewConfig()
	fakeConfig.Clusters["fake"] = api.NewCluster()
	fakeConfig.Clusters["fake"].Server = "https://example.com"
	fakeConfig.Clusters["additional-cluster"] = api.NewCluster()
	fakeConfig.AuthInfos["fake"] = api.NewAuthInfo()
	fakeConfig.AuthInfos["additional-auth"] = api.NewAuthInfo()
	if rc != nil {
		fakeConfig.Clusters["fake"].Server = rc.Host
		fakeConfig.Clusters["fake"].CertificateAuthorityData = rc.TLSClientConfig.CAData
		fakeConfig.AuthInfos["fake"].ClientKeyData = rc.TLSClientConfig.KeyData
		fakeConfig.AuthInfos["fake"].ClientCertificateData = rc.TLSClientConfig.CertData
	}
	fakeConfig.Contexts["fake-context"] = api.NewContext()
	fakeConfig.Contexts["fake-context"].Cluster = "fake"
	fakeConfig.Contexts["fake-context"].AuthInfo = "fake"
	fakeConfig.Contexts["additional-context"] = api.NewContext()
	fakeConfig.Contexts["additional-context"].Cluster = "additional-cluster"
	fakeConfig.Contexts["additional-context"].AuthInfo = "additional-auth"
	fakeConfig.CurrentContext = "fake-context"
	kubeConfig := filepath.Join(c.tempDir, "config")
	_ = clientcmd.WriteToFile(*fakeConfig, kubeConfig)
	_ = os.Setenv("KUBECONFIG", kubeConfig)
	if c.mcpServer != nil {
		if err := c.mcpServer.reloadKubernetesClient(); err != nil {
			panic(err)
		}
	}
	return fakeConfig
}

// withEnvTest sets up the environment for kubeconfig to be used with envTest
func (c *mcpContext) withEnvTest() {
	c.withKubeConfig(envTestRestConfig)
}

// inOpenShift sets up the kubernetes environment to seem to be running OpenShift
func (c *mcpContext) inOpenShift() func() {
	c.withKubeConfig(envTestRestConfig)
	crdTemplate := `
          {
            "apiVersion": "apiextensions.k8s.io/v1",
            "kind": "CustomResourceDefinition",
            "metadata": {"name": "%s"},
            "spec": {
              "group": "%s",
              "versions": [{
                "name": "v1","served": true,"storage": true,
                "schema": {"openAPIV3Schema": {"type": "object","x-kubernetes-preserve-unknown-fields": true}}
              }],
              "scope": "%s",
              "names": {"plural": "%s","singular": "%s","kind": "%s"}
            }
          }`
	removeProjects := c.crdApply(fmt.Sprintf(crdTemplate, "projects.project.openshift.io", "project.openshift.io",
		"Cluster", "projects", "project", "Project"))
	removeRoutes := c.crdApply(fmt.Sprintf(crdTemplate, "routes.route.openshift.io", "route.openshift.io",
		"Namespaced", "routes", "route", "Route"))
	return func() {
		removeProjects()
		removeRoutes()
	}
}

// newKubernetesClient creates a new Kubernetes client with the envTest kubeconfig
func (c *mcpContext) newKubernetesClient() *kubernetes.Clientset {
	return kubernetes.NewForConfigOrDie(envTestRestConfig)
}

func (c *mcpContext) newRestClient(groupVersion *schema.GroupVersion) *rest.RESTClient {
	config := *envTestRestConfig
	config.GroupVersion = groupVersion
	config.APIPath = "/api"
	config.NegotiatedSerializer = serializer.NewCodecFactory(scale.NewScaleConverter().Scheme()).WithoutConversion()
	rc, err := rest.RESTClientFor(&config)
	if err != nil {
		panic(err)
	}
	return rc
}

// newApiExtensionsClient creates a new ApiExtensions client with the envTest kubeconfig
func (c *mcpContext) newApiExtensionsClient() *apiextensionsv1.ApiextensionsV1Client {
	return apiextensionsv1.NewForConfigOrDie(envTestRestConfig)
}

// crdApply creates a CRD from the provided resource string and waits for it to be established, returns a cleanup function
func (c *mcpContext) crdApply(resource string) func() {
	apiExtensionsV1Client := c.newApiExtensionsClient()
	var crd = &apiextensionsv1spec.CustomResourceDefinition{}
	err := json.Unmarshal([]byte(resource), crd)
	_, err = apiExtensionsV1Client.CustomResourceDefinitions().Create(c.ctx, crd, metav1.CreateOptions{})
	if err != nil {
		panic(fmt.Errorf("failed to create CRD %v", err))
	}
	c.crdWaitUntilReady(crd.Name)
	return func() {
		err = apiExtensionsV1Client.CustomResourceDefinitions().Delete(c.ctx, crd.Name, metav1.DeleteOptions{
			GracePeriodSeconds: ptr.To(int64(0)),
		})
		iteration := 0
		for iteration < 10 {
			if _, derr := apiExtensionsV1Client.CustomResourceDefinitions().Get(c.ctx, crd.Name, metav1.GetOptions{}); derr != nil {
				break
			}
			time.Sleep(50 * time.Millisecond)
			iteration++
		}
		if err != nil {
			panic(fmt.Errorf("failed to delete CRD %v", err))
		}
	}
}

// crdWaitUntilReady waits for a CRD to be established
func (c *mcpContext) crdWaitUntilReady(name string) {
	watcher, err := c.newApiExtensionsClient().CustomResourceDefinitions().Watch(c.ctx, metav1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	})
	_, err = toolswatch.UntilWithoutRetry(c.ctx, watcher, func(event watch.Event) (bool, error) {
		for _, c := range event.Object.(*apiextensionsv1spec.CustomResourceDefinition).Status.Conditions {
			if c.Type == apiextensionsv1spec.Established && c.Status == apiextensionsv1spec.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		panic(fmt.Errorf("failed to wait for CRD %v", err))
	}
}

// callTool helper function to call a tool by name with arguments
func (c *mcpContext) callTool(name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	callToolRequest := mcp.CallToolRequest{}
	callToolRequest.Params.Name = name
	callToolRequest.Params.Arguments = args
	return c.mcpClient.CallTool(c.ctx, callToolRequest)
}

func restoreAuth(ctx context.Context) {
	kubernetesAdmin := kubernetes.NewForConfigOrDie(envTest.Config)
	// Authorization
	_, _ = kubernetesAdmin.RbacV1().ClusterRoles().Update(ctx, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: "allow-all"},
		Rules: []rbacv1.PolicyRule{{
			Verbs:     []string{"*"},
			APIGroups: []string{"*"},
			Resources: []string{"*"},
		}},
	}, metav1.UpdateOptions{})
	_, _ = kubernetesAdmin.RbacV1().ClusterRoleBindings().Update(ctx, &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "allow-all"},
		Subjects:   []rbacv1.Subject{{Kind: "Group", Name: envTestUser.Groups[0]}},
		RoleRef:    rbacv1.RoleRef{Kind: "ClusterRole", Name: "allow-all"},
	}, metav1.UpdateOptions{})
}

func createTestData(ctx context.Context) {
	kubernetesAdmin := kubernetes.NewForConfigOrDie(envTestRestConfig)
	// Namespaces
	_, _ = kubernetesAdmin.CoreV1().Namespaces().
		Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns-1"}}, metav1.CreateOptions{})
	_, _ = kubernetesAdmin.CoreV1().Namespaces().
		Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns-2"}}, metav1.CreateOptions{})
	_, _ = kubernetesAdmin.CoreV1().Namespaces().
		Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns-to-delete"}}, metav1.CreateOptions{})
	_, _ = kubernetesAdmin.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "a-pod-in-default"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
	}, metav1.CreateOptions{})
	// Pods for listing
	_, _ = kubernetesAdmin.CoreV1().Pods("ns-1").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "a-pod-in-ns-1"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
	}, metav1.CreateOptions{})
	_, _ = kubernetesAdmin.CoreV1().Pods("ns-2").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "a-pod-in-ns-2"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
	}, metav1.CreateOptions{})
	_, _ = kubernetesAdmin.CoreV1().ConfigMaps("default").
		Create(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "a-configmap-to-delete"}}, metav1.CreateOptions{})
}
