package mcp

import (
	"context"
	"log/slog"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/vibeus/mcp/jsonrpc2"
)

type testServerImpl struct {
	prompts_ListChanged chan struct{}
	prompts_started     sync.Once
}

func NewTestServerImpl() *testServerImpl {
	return &testServerImpl{}
}

func (*testServerImpl) NegotiateMCPVersion(client string) string {
	return LatestMCPVersion
}
func (c *testServerImpl) Prompts_Started() *sync.Once {
	return &c.prompts_started
}
func (c *testServerImpl) Prompts_Capability() *CapPrompts {
	return &CapPrompts{ListChanged: c.prompts_ListChanged != nil}
}
func (c *testServerImpl) Prompts_OnList() []PagedPrompts     { return []PagedPrompts{} }
func (c *testServerImpl) Prompts_ListChanged() chan struct{} { return c.prompts_ListChanged }
func (c *testServerImpl) Prompts_OnGet(name string) (PromptGetResponse, *jsonrpc2.ErrorObject) {
	return PromptGetResponse{}, nil
}

type testClientImpl struct {
	roots_ListChanged chan struct{}
	roots_started     sync.Once
}

func (c *testClientImpl) Roots_Started() *sync.Once {
	return &c.roots_started
}

func (c *testClientImpl) Roots_Capability() *CapRoots {
	return &CapRoots{ListChanged: c.roots_ListChanged != nil}
}

func (c *testClientImpl) Roots_OnList() []Root {
	return []Root{
		{URI: "file://myfile", Name: "Example Root"},
	}
}

func (c *testClientImpl) Roots_ListChanged() chan struct{} {
	return c.roots_ListChanged
}

func (c *testClientImpl) Sampling_Capability() *CapSampling {
	return new(CapSampling)
}

func (c *testClientImpl) HandleRequest(w jsonrpc2.ResponseWriterOf[SamplingResponse], msg SamplingMessage) error {
	res := SamplingResponse{}
	return w.WriteResponse(res)
}

func TestInitialize(t *testing.T) {
	sconn, cconn := net.Pipe()

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slogger := logger.WithGroup("server")
	clogger := logger.WithGroup("client")

	serverProvider := &testServerImpl{prompts_ListChanged: make(chan struct{})}
	server := NewServer(sconn)
	serverInstance := &ServerImpl{MCPVersionNegotiator: serverProvider, CapPromptsProvider: serverProvider}
	server.Setup(serverInstance)
	server.SetLogger(slogger) // Set the logger for the server
	server.SetMCPVersion(LatestMCPVersion)
	server.SetCapabilities(serverInstance.Capabilities())

	clientProvider := &testClientImpl{roots_ListChanged: make(chan struct{})}
	client := NewClient(cconn)
	clientInstance := &ClientImpl{CapRootsProvider: clientProvider, CapSamplingProvider: clientProvider}
	client.Setup(clientInstance)
	client.SetLogger(clogger) // Set the logger for the client
	client.SetMCPVersion(LatestMCPVersion)
	client.SetCapabilities(clientInstance.Capabilities())

	go func() {
		server.Serve()
	}()
	var err error
	/*
		err = client.Ping(context.Background())
		if err != nil {
			t.Fail()
		}
	*/
	err = client.Initialize(context.Background())
	if err != nil {
		t.Fail()
	}
	err = client.Initialized(context.Background())
	if err != nil {
		t.Fail()
	}

	<-time.After(100 * time.Millisecond)
}
