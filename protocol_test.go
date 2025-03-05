package mcp

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"
)

type testServerImpl struct {
}

func (testServerImpl) NegotiateMCPVersion(client string) string {
	return LatestMCPVersion
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

	client := NewClient(cconn)
	client.SetLogger(clogger) // Set the logger for the client
	client.SetMCPVersion(LatestMCPVersion)
	client.SetCapabilities(ClientCapabilities{})

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
