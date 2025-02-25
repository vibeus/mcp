package mcp

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"

	"go.lsp.dev/jsonrpc2"
)

type testServerImpl struct {
}

func (testServerImpl) NegotiateMCPVersion(client string) string {
	return LatestMCPVersion
}

func TestInitialize(t *testing.T) {
	spipe, cpipe := net.Pipe()
	sconn := jsonrpc2.NewConn(jsonrpc2.NewRawStream(spipe))
	cconn := jsonrpc2.NewConn(jsonrpc2.NewRawStream(cpipe))

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slogger := logger.WithGroup("server")
	clogger := logger.WithGroup("client")

	server := NewServer(sconn)
	server.versionNegotiator = testServerImpl{}
	server.SetLogger(slogger) // Set the logger for the server
	server.SetMCPVersion(LatestMCPVersion)
	server.SetCapabilities(ServerCapabilities{})

	client := NewClient(cconn)
	client.SetLogger(clogger) // Set the logger for the client
	client.SetMCPVersion(LatestMCPVersion)
	client.SetCapabilities(ClientCapabilities{})

	go server.ServeStream(context.Background(), sconn)

	var err error
	err = client.Ping(context.Background())
	if err != nil {
		t.Fail()
	}
	err = client.Initialize(context.Background())
	if err != nil {
		t.Fail()
	}
	err = client.Initialized(context.Background())
	if err != nil {
		t.Fail()
	}
}
