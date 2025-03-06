package mcp

import (
	"context"
	"log/slog"
	"net"
	"os"
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

func (c testClientImpl) Sampling_OnCreateMessage(SamplingMessage) (<-chan SamplingResponse, <-chan jsonrpc2.ErrorObject) {
	responseChan := make(chan SamplingResponse)
	errorChan := make(chan jsonrpc2.ErrorObject)
	go func() {
		// Simulate a response from the client
		responseChan <- SamplingResponse{}
	}()
	return responseChan, errorChan
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
	clientInstance := &ClientImpl{CapRootsProvider: clientProvider, CapSamplingProvider: clientProvider}
	client := NewClient(cconn, clientInstance)
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
