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
}

func (testServerImpl) NegotiateMCPVersion(client string) string {
	return LatestMCPVersion
}

type testClientImpl struct {
	roots_ListChanged chan struct{}
	roots_started     sync.Once
}

func (c testClientImpl) Roots_Started() *sync.Once {
	return &c.roots_started
}

func (c testClientImpl) Roots_Capability() *CapRoots {
	return &CapRoots{ListChanged: c.roots_ListChanged != nil}
}

func (c testClientImpl) Roots_OnList() []Root {
	return []Root{
		{URI: "file://myfile", Name: "Example Root"},
	}
}

func (c testClientImpl) Roots_ListChanged() chan struct{} {
	return c.roots_ListChanged
}

func (c testClientImpl) Sampling_Capability() *CapSampling {
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

	server := NewServer(sconn)
	server.versionNegotiator = testServerImpl{}
	server.SetLogger(slogger) // Set the logger for the server
	server.SetMCPVersion(LatestMCPVersion)
	server.SetCapabilities(ServerCapabilities{})

	clientProvider := &testClientImpl{roots_ListChanged: make(chan struct{})}
	client := NewClient(cconn)
	clientInstance := &ClientImpl{CapRootsProvider: clientProvider, CapSamplingProvider: clientProvider}
	client.Start(clientInstance)
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
