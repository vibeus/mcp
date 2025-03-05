package jsonrpc2

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"
)

// TestRequestResponse tests a successful request/response cycle.
func TestRequestResponse(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	clientConn, serverConn := net.Pipe()
	context.AfterFunc(ctx, func() {
		clientConn.Close()
		serverConn.Close()
	})

	// Create client and server
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewPeer(ctx, NewLineFramer(clientConn), nil)
	server := NewPeer(ctx, NewLineFramer(serverConn), &testHandler{})
	client.SetLogger(logger.WithGroup("client"))
	server.SetLogger(logger.WithGroup("server"))

	server.Start()

	// Send a request from the client
	var result string

	req, err := client.Call("testMethod", "testParams")
	if err != nil {
		t.Fatalf("Client call error: %v", err)
	}
	err = client.RecvResponse(*req, &result)
	if err != nil {
		t.Fatalf("Client RecvResponse error: %v", err)
	}

	// Check the response
	if result != "testResponse" {
		t.Fatalf("Unexpected response value: %v", result)
	}
}

// TestRequestError tests a request that results in an error response.
func TestRequestError(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	clientConn, serverConn := net.Pipe()
	context.AfterFunc(ctx, func() {
		clientConn.Close()
		serverConn.Close()
	})

	// Create client and server
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewPeer(ctx, NewLineFramer(clientConn), nil)
	server := NewPeer(ctx, NewLineFramer(serverConn), &testHandler{})
	client.SetLogger(logger.WithGroup("client"))
	server.SetLogger(logger.WithGroup("server"))

	server.Start()

	// Send a request from the client that will trigger an error
	var result any
	req, err := client.Call("errorMethod", "errorParams")
	if err != nil {
		t.Fatalf("Client call error: %v", err)
	}
	err = client.RecvResponse(*req, &result)
	if err == nil {
		t.Fatalf("Expected error in response, got nil")
	}

	// Check the error
	if err.Error() != "jsonrpc2 error code -32603: Internal error\n null" {
		t.Fatalf("Unexpected error message: %v", err)
	}
}

// TestNotification tests a notification (request without ID).
func TestNotification(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	clientConn, serverConn := net.Pipe()
	context.AfterFunc(ctx, func() {
		clientConn.Close()
		serverConn.Close()
	})

	// Create client and server
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewPeer(ctx, NewLineFramer(clientConn), nil)
	server := NewPeer(ctx, NewLineFramer(serverConn), &testHandler{})
	client.SetLogger(logger.WithGroup("client"))
	server.SetLogger(logger.WithGroup("server"))

	server.Start()

	// Send a notification from the client
	err := client.Notify("notifyMethod", "notifyParams")
	if err != nil {
		t.Fatalf("Client notify error: %v", err)
	}

	// Wait for the server to process the notification
	time.Sleep(100 * time.Millisecond)
}

// testHandler is a simple handler for testing purposes.
type testHandler struct{}

func (h *testHandler) HandleRequest(w ResponseWriter, req Request) error {
	switch req.Method {
	case "testMethod":
		return w.WriteResponse("testResponse")
	case "errorMethod":
		return w.WriteError(ErrorObject{Code: JSONRPC2ErrorInternalError, Message: "Internal error"})
	case "notifyMethod":
		var param string
		err := json.Unmarshal(*req.Params, &param)
		if err != nil {
			return err
		}
		if param != "notifyParams" {
			return fmt.Errorf("Invalid parameter: %s", param)
		}
		return nil
	default:
		return w.WriteError(ErrorObject{Code: JSONRPC2ErrorMethodNotFound, Message: "Method not found"})
	}
}

// TestCancel verifies that canceling a pending request results in the expected error
// and cleans up the pending request entry.
func TestCancel(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	clientConn, serverConn := net.Pipe()
	context.AfterFunc(ctx, func() {
		clientConn.Close()
		serverConn.Close()
	})

	// Create client and server with a slow handler
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewPeer(ctx, NewLineFramer(clientConn), nil)
	server := NewPeer(ctx, NewLineFramer(serverConn), &slowTestHandler{})
	client.SetLogger(logger.WithGroup("client"))
	server.SetLogger(logger.WithGroup("server"))

	server.Start()

	// Send a request that the server will process slowly
	req, err := client.Call("slowMethod", nil)
	if err != nil {
		t.Fatalf("Client.Call error: %v", err)
	}

	// Cancel the request before the server can respond
	client.Cancel(*req)

	// Attempt to receive the response, expecting cancellation
	var result string
	err = client.RecvResponse(*req, &result)
	if err == nil {
		t.Fatal("Expected error due to cancellation, got nil")
	}

	rpcErr, ok := err.(RPCError)
	if !ok {
		t.Fatalf("Expected RPCError, got %T: %v", err, err)
	}
	if rpcErr.Unwrap() != ErrContextCancel {
		t.Fatalf("Expected error %v, got %v", ErrContextCancel, rpcErr.Unwrap())
	}

	// Verify the pending request is removed
	client.mutex.Lock()
	_, exists := client.pendingRequests[req.id]
	client.mutex.Unlock()
	if exists {
		t.Error("Pending request was not cleaned up after cancellation")
	}
}

// slowTestHandler is a handler that sleeps to simulate processing delay.
type slowTestHandler struct{}

func (h *slowTestHandler) HandleRequest(w ResponseWriter, req Request) error {
	if req.Method == "slowMethod" {
		time.Sleep(500 * time.Millisecond) // Simulate slow processing
		return w.WriteResponse("response")
	}
	return w.WriteError(ErrorObject{Code: JSONRPC2ErrorMethodNotFound, Message: "Method not found"})
}
