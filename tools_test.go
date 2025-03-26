package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/vibeus/mcp/jsonrpc2"
)

func TestToolsCapability(t *testing.T) {
	// Setup test environment
	serverProvider := NewTestServerImpl()
	serverInstance := &ServerImpl{
		MCPVersionNegotiator: serverProvider,
		CapPromptsProvider:   serverProvider,
		CapToolsProvider:     serverProvider,
	}
	clientProvider := &testClientImpl{roots_ListChanged: make(chan struct{})}
	clientInstance := &ClientImpl{
		CapRootsProvider:    clientProvider,
		CapSamplingProvider: clientProvider,
	}

	ts, err := SetupClientServer(serverInstance, clientInstance)
	if err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	defer ts.Cleanup()

	// Run test cases
	t.Run("Initialization", func(t *testing.T) {
		ts.Init(t)
	})

	// Run tools-specific tests
	t.Run("ToolsList", func(t *testing.T) {
		tools, err := ts.Client.ToolsList(ts.Ctx, "")
		if err != nil {
			t.Fatalf("ListTools failed: %v", err)
		}
		if len(tools) == 0 {
			t.Error("Expected at least one tool, got none")
		}
	})

	t.Run("ToolsNotifications", func(t *testing.T) {
		// Create a channel to track notification completion
		notifyDone := make(chan struct{})

		go func() {
			// Trigger a tools list change
			serverProvider.tools_ListChanged <- struct{}{}
			close(notifyDone)
		}()

		select {
		case <-notifyDone:
			// Notification sent successfully
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for tools list change notification")
		}
	})

	// Run tool-specific tests
	t.Run("ToolCall", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ts.Ctx, 1*time.Second)
		defer cancel()

		// Test with valid tool and parameters
		response, err := ts.Client.ToolCall(ctx, "test_tool", map[string]string{"param1": "value1"})
		if err != nil {
			t.Fatalf("ToolCall failed: %v", err)
		}
		if response.IsError {
			t.Error("Expected successful tool execution")
		}
		if len(response.Content) == 0 {
			t.Error("Expected non-empty response content")
		}
		if response.Content[0].Type != "text" || response.Content[0].Text != "Tool executed successfully" {
			t.Errorf("Unexpected response content: %v", response.Content[0])
		}

		// Test with invalid tool
		_, err = ts.Client.ToolCall(ctx, "nonexistent_tool", map[string]string{})
		if err == nil {
			t.Error("Expected error for non-existent tool")
		}
		rpcErr, ok := err.(*jsonrpc2.ErrorObject)
		if !ok {
			t.Fatalf("Expected jsonrpc2.ErrorObject, got %T", err)
		}
		if rpcErr.Code != -32601 {
			t.Errorf("Expected code -32601 (MethodNotFound), got %d", rpcErr.Code)
		}
	})

	ts.Cancel()
}
