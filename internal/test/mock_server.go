package test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
)

type MockServer struct {
	server       *httptest.Server
	config       *rest.Config
	restHandlers []http.HandlerFunc
}

func NewMockServer() *MockServer {
	ms := &MockServer{}
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	ms.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		for _, handler := range ms.restHandlers {
			handler(w, req)
		}
	}))
	ms.config = &rest.Config{
		Host:    ms.server.URL,
		APIPath: "/api",
		ContentConfig: rest.ContentConfig{
			NegotiatedSerializer: codecs,
			ContentType:          runtime.ContentTypeJSON,
			GroupVersion:         &v1.SchemeGroupVersion,
		},
	}
	ms.restHandlers = make([]http.HandlerFunc, 0)
	return ms
}

func (m *MockServer) Close() {
	m.server.Close()
}

func (m *MockServer) Handle(handler http.Handler) {
	m.restHandlers = append(m.restHandlers, handler.ServeHTTP)
}

func (m *MockServer) Config() *rest.Config {
	return m.config
}

func (m *MockServer) KubeConfig() *api.Config {
	fakeConfig := api.NewConfig()
	fakeConfig.Clusters["fake"] = api.NewCluster()
	fakeConfig.Clusters["fake"].Server = m.config.Host
	fakeConfig.Clusters["fake"].CertificateAuthorityData = m.config.CAData
	fakeConfig.AuthInfos["fake"] = api.NewAuthInfo()
	fakeConfig.AuthInfos["fake"].ClientKeyData = m.config.KeyData
	fakeConfig.AuthInfos["fake"].ClientCertificateData = m.config.CertData
	fakeConfig.Contexts["fake-context"] = api.NewContext()
	fakeConfig.Contexts["fake-context"].Cluster = "fake"
	fakeConfig.Contexts["fake-context"].AuthInfo = "fake"
	fakeConfig.CurrentContext = "fake-context"
	return fakeConfig
}

func WriteObject(w http.ResponseWriter, obj runtime.Object) {
	w.Header().Set("Content-Type", runtime.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(obj); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type streamAndReply struct {
	httpstream.Stream
	replySent <-chan struct{}
}

type StreamContext struct {
	Closer       io.Closer
	StdinStream  io.ReadCloser
	StdoutStream io.WriteCloser
	StderrStream io.WriteCloser
	writeStatus  func(status *apierrors.StatusError) error
}

type StreamOptions struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func v4WriteStatusFunc(stream io.Writer) func(status *apierrors.StatusError) error {
	return func(status *apierrors.StatusError) error {
		bs, err := json.Marshal(status.Status())
		if err != nil {
			return err
		}
		_, err = stream.Write(bs)
		return err
	}
}
func CreateHTTPStreams(w http.ResponseWriter, req *http.Request, opts *StreamOptions) (*StreamContext, error) {
	_, err := httpstream.Handshake(req, w, []string{"v4.channel.k8s.io"})
	if err != nil {
		return nil, err
	}

	upgrader := spdy.NewResponseUpgrader()
	streamCh := make(chan streamAndReply)
	connection := upgrader.UpgradeResponse(w, req, func(stream httpstream.Stream, replySent <-chan struct{}) error {
		streamCh <- streamAndReply{Stream: stream, replySent: replySent}
		return nil
	})
	ctx := &StreamContext{
		Closer: connection,
	}

	// wait for stream
	replyChan := make(chan struct{}, 4)
	defer close(replyChan)
	receivedStreams := 0
	expectedStreams := 1
	if opts.Stdout != nil {
		expectedStreams++
	}
	if opts.Stdin != nil {
		expectedStreams++
	}
	if opts.Stderr != nil {
		expectedStreams++
	}
WaitForStreams:
	for {
		select {
		case stream := <-streamCh:
			streamType := stream.Headers().Get(v1.StreamType)
			switch streamType {
			case v1.StreamTypeError:
				replyChan <- struct{}{}
				ctx.writeStatus = v4WriteStatusFunc(stream)
			case v1.StreamTypeStdout:
				replyChan <- struct{}{}
				ctx.StdoutStream = stream
			case v1.StreamTypeStdin:
				replyChan <- struct{}{}
				ctx.StdinStream = stream
			case v1.StreamTypeStderr:
				replyChan <- struct{}{}
				ctx.StderrStream = stream
			default:
				// add other stream ...
				return nil, errors.New("unimplemented stream type")
			}
		case <-replyChan:
			receivedStreams++
			if receivedStreams == expectedStreams {
				break WaitForStreams
			}
		}
	}

	return ctx, nil
}
