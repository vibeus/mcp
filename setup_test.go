package mcp

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"
)

// TestSetup contains the client and server instances for testing
type TestSetup struct {
	ServerConn net.Conn
	ClientConn net.Conn
	Server     *ServerState
	Client     *ClientState
	Cancel     context.CancelFunc
	Ctx        context.Context
}

// SetupClientServer creates and initializes client and server instances for testing
func SetupClientServer(
	serverProvider ServerProvider,
	clientProvider ClientProvider,
) (*TestSetup, error) {
	// Create pipe connection
	sconn, cconn := net.Pipe()

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slogger := logger.WithGroup("server")
	clogger := logger.WithGroup("client")

	// Setup server
	server := NewServer(sconn)
	server.Setup(serverProvider)
	server.SetLogger(slogger)
	server.SetMCPVersion(LatestMCPVersion)
	server.SetCapabilities(serverProvider.Capabilities())

	// Setup client
	client := NewClient(cconn)
	client.Setup(clientProvider)
	client.SetLogger(clogger)
	client.SetMCPVersion(LatestMCPVersion)
	client.SetCapabilities(clientProvider.Capabilities())

	// Start server in background
	go func() {
		if err := server.Serve(); err != nil {
			slogger.Error("Server stopped", "error", err)
		}
	}()

	return &TestSetup{
		ServerConn: sconn,
		ClientConn: cconn,
		Server:     server,
		Client:     client,
		Cancel:     cancel,
		Ctx:        ctx,
	}, nil
}

// Cleanup closes connections and cancels context
func (ts *TestSetup) Cleanup() {
	ts.Cancel()
	ts.ServerConn.Close()
	ts.ClientConn.Close()
}

func (ts *TestSetup) Init(t *testing.T) {
	// Initialize connection
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := ts.Client.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := ts.Client.Initialized(ctx); err != nil {
		t.Fatalf("Initialized failed: %v", err)
	}
}
