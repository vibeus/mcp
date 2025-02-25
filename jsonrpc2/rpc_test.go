package jsonrpc2

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestRequestResponse tests a successful request/response cycle.
func TestRequestResponse(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	// Create client and server
	client := NewClient(context.Background(), clientConn, NewLineFramer(clientConn))
	server := NewServer(context.Background(), serverConn, NewLineFramer(serverConn), &testHandler{})

	// Start the server
	go func() {
		if err := server.Serve(); err != nil {
			t.Errorf("Server error: %v", err)
		}
	}()

	// Send a request from the client
	var result result

	err := client.Call(MakeNumberID(1), "testMethod", "testParams", &result)
	if err != nil {
		t.Fatalf("Client call error: %v", err)
	}

	// Check the response
	if result.Error != nil {
		t.Fatalf("Unexpected error in response: %v", result.Error)
	}
	if result.Value != "testResponse" {
		t.Fatalf("Unexpected response value: %v", result.Value)
	}
}

// TestRequestError tests a request that results in an error response.
func TestRequestError(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	// Create client and server
	client := NewClient(context.Background(), clientConn, NewLineFramer(clientConn))
	server := NewServer(context.Background(), serverConn, NewLineFramer(serverConn), &testHandler{})

	// Start the server
	go func() {
		if err := server.Serve(); err != nil {
			t.Errorf("Server error: %v", err)
		}
	}()

	// Send a request from the client that will trigger an error
	var result result
	err := client.Call(MakeNumberID(2), "errorMethod", "errorParams", &result)
	if err != nil {
		t.Fatalf("Expected nil error")
	}

	// Check the error
	if result.Error == nil {
		t.Fatalf("Expected error in response, got nil")
	}
	if result.Error.Error() != "jsonrpc2 error code -32603: Internal error\n null" {
		t.Fatalf("Unexpected error message: %v", result.Error)
	}
}

// TestNotification tests a notification (request without ID).
func TestNotification(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	// Create client and server
	client := NewClient(context.Background(), clientConn, NewLineFramer(clientConn))
	server := NewServer(context.Background(), serverConn, NewLineFramer(serverConn), &testHandler{})

	// Start the server
	go func() {
		if err := server.Serve(); err != nil {
			t.Errorf("Server error: %v", err)
		}
	}()

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
		return w.WriteResponse(ResponseOk{Result: "testResponse"})
	case "errorMethod":
		return w.WriteError(ResponseError{Error: ErrorObject{Code: JSONRPC2ErrorInternalError, Message: "Internal error"}})
	default:
		return w.WriteError(ResponseError{Error: ErrorObject{Code: JSONRPC2ErrorMethodNotFound, Message: "Method not found"}})
	}
}

func (h *testHandler) HandleNotification(req Request) error {
	if req.Method != "notifyMethod" {
		return nil
	}
	return nil
}
