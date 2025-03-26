package mcp

import (
	"testing"
	"time"

	"github.com/vibeus/mcp/jsonrpc2"
)

func TestResourcesCapability(t *testing.T) {
	// Setup test environment
	serverProvider := NewTestServerImpl()
	serverInstance := &ServerImpl{
		MCPVersionNegotiator: serverProvider,
		CapPromptsProvider:   serverProvider,
		CapToolsProvider:     serverProvider,
		CapResourcesProvider: serverProvider,
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

	// Run resources-specific tests
	t.Run("ListResources", func(t *testing.T) {
		response, err := ts.Client.ResourcesList(ts.Ctx, "")
		if err != nil {
			t.Fatalf("ResourcesList failed: %v", err)
		}
		if len(response.Resources) == 0 {
			t.Error("Expected at least one test resource")
		}
	})

	t.Run("ListResourceTemplates", func(t *testing.T) {
		templates, err := ts.Client.ResourcesTemplatesList(ts.Ctx)
		if err != nil {
			t.Fatalf("ResourcesList failed: %v", err)
		}
		if len(templates) == 0 {
			t.Error("Expected at least one test resource")
		}
	})

	t.Run("GetResource", func(t *testing.T) {
		resource, err := ts.Client.ResourcesRead(ts.Ctx, "resource://test/0")
		if err != nil {
			t.Fatalf("ResourcesGet failed: %v", err)
		}
		if len(resource) == 0 {
			t.Error("Expected non-empty resource content")
		}
	})

	t.Run("GetNonexistentResource", func(t *testing.T) {
		_, err := ts.Client.ResourcesRead(ts.Ctx, "bad_resource")
		if err == nil {
			t.Fatal("Expected error for non-existent resource")
		}
		rpcErr, ok := err.(*jsonrpc2.ErrorObject)
		if !ok {
			t.Fatalf("Expected jsonrpc2.ErrorObject, got %T", err)
		}
		if rpcErr.Code != JSONRPC2ResourceNotFound {
			t.Errorf("Expected code -32002 (ResourceNotFound), got %d", rpcErr.Code)
		}
	})

	t.Run("ResourcesNotifications", func(t *testing.T) {
		// Create a channel to track notification completion
		notifyDone := make(chan struct{})

		go func() {
			// Trigger a resources list change
			serverProvider.resources_ListChanged <- struct{}{}
			close(notifyDone)
		}()

		select {
		case <-notifyDone:
			// Notification sent successfully
		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for resources list change notification")
		}
	})

	// Cleanup
	ts.Cancel()
	<-time.After(100 * time.Millisecond) // Allow for graceful shutdown
}
